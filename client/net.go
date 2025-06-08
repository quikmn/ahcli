// FILE: client/net.go
package main

import (
	"ahcli/common/logger"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"ahcli/common"
)

var (
	currentChannel string
	cryptoReady    bool
)

func connectToServer(config *ClientConfig) error {
	target := config.Servers[config.PreferredServer].IP
	logger.Info("Resolving server address: %s", target)

	raddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		logger.Error("Failed to resolve UDP address %s: %v", target, err)
		return err
	}

	logger.Info("Establishing UDP connection to %s", raddr)
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		logger.Error("Failed to dial UDP connection: %v", err)
		return err
	}
	defer conn.Close()

	// Send connect request
	req := common.ConnectRequest{
		Type:     "connect",
		Nicklist: config.Nickname,
	}
	data, _ := json.Marshal(req)
	logger.Info("Sending connection request with nicknames: %v", config.Nickname)
	conn.Write(data)

	// Wait for response
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		logger.Error("Connection timeout or error: %v", err)
		return err
	}

	var resp map[string]interface{}
	json.Unmarshal(buffer[:n], &resp)

	switch resp["type"] {
	case "accept":
		var accepted common.ConnectAccepted
		json.Unmarshal(buffer[:n], &accepted)

		currentChannel = "General" // Default channel

		appState.SetConnected(true, accepted.Nickname, accepted.ServerName, accepted.MOTD)
		appState.SetChannel(currentChannel)
		appState.SetChannels(accepted.Channels)

		// Initialize channel users - put all users in the default channel for now
		channelUsers := make(map[string][]string)
		for _, channel := range accepted.Channels {
			channelUsers[channel] = make([]string, 0)
		}
		// Put all users in the default channel initially
		if len(accepted.Channels) > 0 {
			channelUsers[currentChannel] = accepted.Users
		}

		appState.SetChannelUsers(channelUsers)

		logger.Info("Connected as: %s", accepted.Nickname)
		logger.Info("MOTD: %s", accepted.MOTD)
		logger.Info("Available channels: %v", accepted.Channels)
		logger.Info("Current users: %v", accepted.Users)

		// Initiate crypto handshake after successful connection
		err = initiateCryptoHandshake(conn)
		if err != nil {
			logger.Error("Crypto handshake failed: %v", err)
			appState.AddMessage("Warning: Chat encryption unavailable", "warning")
		}

	case "reject":
		var reject common.Reject
		json.Unmarshal(buffer[:n], &reject)
		logger.Error("Connection rejected: %s", reject.Message)
		return fmt.Errorf("connection rejected: %s", reject.Message)
	default:
		logger.Error("Unexpected response type: %v", resp["type"])
		return fmt.Errorf("unexpected response type: %v", resp["type"])
	}

	conn.SetReadDeadline(time.Time{})
	serverConn = conn

	go handleServerResponses(conn)
	go startPingLoop(conn)

	select {}
}

func initiateCryptoHandshake(conn *net.UDPConn) error {
	logger.Info("Initiating crypto handshake with server")

	// Get client public key
	clientPubKey := clientCrypto.GetPublicKey()

	// Send handshake request
	handshake := map[string]string{
		"type":       "crypto_handshake",
		"public_key": base64.StdEncoding.EncodeToString(clientPubKey[:]),
	}

	data, err := json.Marshal(handshake)
	if err != nil {
		logger.Error("Failed to marshal crypto handshake: %v", err)
		return fmt.Errorf("failed to marshal handshake: %v", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		logger.Error("Failed to send crypto handshake: %v", err)
		return fmt.Errorf("failed to send handshake: %v", err)
	}

	logger.Debug("Crypto handshake request sent, waiting for response")

	// Wait for handshake response with timeout
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		logger.Error("Crypto handshake timeout: %v", err)
		return fmt.Errorf("handshake timeout: %v", err)
	}

	var response struct {
		Type      string `json:"type"`
		Status    string `json:"status"`
		PublicKey string `json:"public_key"`
		Error     string `json:"error"`
	}

	err = json.Unmarshal(buffer[:n], &response)
	if err != nil {
		logger.Error("Invalid crypto handshake response: %v", err)
		return fmt.Errorf("invalid handshake response: %v", err)
	}

	if response.Type != "crypto_handshake_response" {
		logger.Error("Unexpected handshake response type: %s", response.Type)
		return fmt.Errorf("unexpected response type: %s", response.Type)
	}

	if response.Status != "success" {
		logger.Error("Crypto handshake failed: %s", response.Error)
		return fmt.Errorf("handshake failed: %s", response.Error)
	}

	// Decode server public key
	serverPubKeyBytes, err := base64.StdEncoding.DecodeString(response.PublicKey)
	if err != nil {
		logger.Error("Invalid server public key format: %v", err)
		return fmt.Errorf("invalid server public key: %v", err)
	}

	if len(serverPubKeyBytes) != 32 {
		logger.Error("Invalid server public key length: %d bytes", len(serverPubKeyBytes))
		return fmt.Errorf("invalid server public key length: %d", len(serverPubKeyBytes))
	}

	var serverPubKey [32]byte
	copy(serverPubKey[:], serverPubKeyBytes)

	// Complete the handshake
	err = clientCrypto.CompleteHandshake(serverPubKey)
	if err != nil {
		logger.Error("Failed to complete crypto handshake: %v", err)
		return fmt.Errorf("failed to complete handshake: %v", err)
	}

	cryptoReady = true
	appState.AddMessage("🔒 Chat encryption enabled", "success")
	logger.Info("Crypto handshake completed successfully - E2E encryption active")

	// Clear the read deadline
	conn.SetReadDeadline(time.Time{})

	return nil
}

// Called from Web UI
func changeChannel(channel string) {
	if serverConn == nil {
		logger.Error("Cannot change channel: not connected to server")
		return
	}

	change := map[string]string{
		"type":    "change_channel",
		"channel": channel,
	}
	data, _ := json.Marshal(change)
	serverConn.Write(data)

	logger.Info("Requested channel switch to: %s", channel)
}

// Send chat message to server - now with encryption support
func sendChatMessage(message string) {
	if serverConn == nil {
		logger.Error("Cannot send chat: not connected to server")
		appState.AddMessage("Cannot send chat: not connected", "error")
		return
	}

	if currentChannel == "" {
		logger.Error("Cannot send chat: no current channel")
		appState.AddMessage("Cannot send chat: no channel", "error")
		return
	}

	// Get current user nickname
	state := appState.GetState()
	nickname := state["nickname"].(string)

	logger.Info("Attempting to send chat message: %s", message)

	// Try encrypted chat first if crypto is ready
	if cryptoReady && clientCrypto.IsReady() {
		err := sendEncryptedChatMessage(message, nickname)
		if err != nil {
			logger.Error("Encrypted chat failed, falling back to plaintext: %v", err)
			appState.AddMessage("Encryption failed, sent as plaintext", "warning")
			// Fall through to plaintext
		} else {
			logger.Info("✅ Sent encrypted chat message: %s", message)
			return
		}
	}

	// Fallback to plaintext chat
	chatMsg := map[string]string{
		"type":     "chat",
		"channel":  currentChannel,
		"message":  message,
		"username": nickname,
	}

	data, err := json.Marshal(chatMsg)
	if err != nil {
		logger.Error("Failed to marshal chat message: %v", err)
		return
	}

	_, err = serverConn.Write(data)
	if err != nil {
		logger.Error("Failed to send chat message: %v", err)
		appState.AddMessage("Failed to send chat message", "error")
	} else {
		logger.Info("✅ Sent plaintext chat message: %s", message)
	}
}

func sendEncryptedChatMessage(message, username string) error {
	logger.Debug("Encrypting chat message for transmission")

	// Encrypt the message
	encryptedData, err := clientCrypto.EncryptMessage(message)
	if err != nil {
		return fmt.Errorf("encryption failed: %v", err)
	}

	// Create encrypted chat message
	encryptedMsg := map[string]interface{}{
		"type":      "encrypted_chat",
		"channel":   currentChannel,
		"encrypted": true,
		"payload":   base64.StdEncoding.EncodeToString(encryptedData),
	}

	data, err := json.Marshal(encryptedMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted message: %v", err)
	}

	_, err = serverConn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send encrypted message: %v", err)
	}

	logger.Debug("Encrypted chat message sent successfully")
	return nil
}

func handleServerResponses(conn *net.UDPConn) {
	logger.Info("Starting server response handler")

	buffer := make([]byte, 4096)
	var networkFrameCount int
	var lastSeqNum uint16 = 0
	var packetsReceived int
	var packetsLost int

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			logger.Error("Disconnected from server: %v", err)
			appState.SetConnected(false, "", "", "")
			appState.AddMessage("Disconnected from server", "error")
			cryptoReady = false // Reset crypto state on disconnect
			return
		}

		// Try to parse JSON first (control messages including chat)
		var msg map[string]interface{}
		if err := json.Unmarshal(buffer[:n], &msg); err == nil {
			switch msg["type"] {
			case "channel_changed":
				channelName := msg["channel"].(string)
				currentChannel = channelName

				appState.SetChannel(channelName)
				logger.Info("Channel changed to: %s", channelName)

			case "error":
				errorMsg := msg["message"].(string)
				appState.AddMessage(fmt.Sprintf("Server error: %s", errorMsg), "error")
				logger.Error("Server error: %s", errorMsg)

			case "pong":
				logger.Debug("Received pong from server")

			case "channel_users_update":
				var update struct {
					ChannelUsers map[string][]string `json:"channelUsers"`
				}
				if err := json.Unmarshal(buffer[:n], &update); err == nil {
					appState.SetChannelUsers(update.ChannelUsers)
					logger.Debug("Channel users updated")
				}

			case "chat_message":
				logger.Info("Received chat message from server")
				handleIncomingChatMessage(buffer[:n])

			case "encrypted_chat":
				logger.Info("Received encrypted chat message from server")
				handleIncomingEncryptedChatMessage(buffer[:n])

			case "chat_history":
				logger.Info("Received chat history from server")
				handleChatHistory(buffer[:n])

			default:
				logger.Debug("Unknown server message type: %v", msg["type"])
			}
			continue
		}

		// Not JSON, try premium audio packet
		if n < 6 { // Minimum: 2 bytes prefix + 2 bytes seq + 2 bytes audio
			logger.Debug("Dropped malformed packet (too small): %d bytes", n)
			continue
		}

		// Validate audio packet prefix
		prefix := binary.LittleEndian.Uint16(buffer[0:2])
		if prefix != 0x5541 { // 'AU'
			logger.Debug("Dropped packet with invalid prefix: 0x%04X", prefix)
			continue
		}

		// Extract sequence number (premium packets)
		seqNum := binary.LittleEndian.Uint16(buffer[2:4])

		// Calculate audio payload size
		sampleCount := (n - 4) / 2 // Skip 4 bytes (prefix + seq), 2 bytes per sample
		if sampleCount != framesPerBuffer {
			logger.Debug("Dropped frame with wrong length: got %d samples, expected %d", sampleCount, framesPerBuffer)
			continue
		}

		// Decode audio samples
		samples := make([]int16, sampleCount)
		err = binary.Read(bytes.NewReader(buffer[4:n]), binary.LittleEndian, &samples)
		if err != nil {
			logger.Error("Failed to decode audio samples: %v", err)
			continue
		}

		// Track packet statistics for network quality
		packetsReceived++
		if packetsReceived > 1 { // Skip first packet for sequence analysis
			expectedSeq := lastSeqNum + 1
			if seqNum != expectedSeq {
				if seqNum > expectedSeq {
					// Packets were lost
					lost := int(seqNum - expectedSeq)
					packetsLost += lost
					logger.Debug("Packet loss detected: expected %d, got %d (%d packets lost)",
						expectedSeq, seqNum, lost)
				} else {
					// Out of order packet (late arrival)
					logger.Debug("Out-of-order packet: expected %d, got %d", expectedSeq, seqNum)
				}
			}
		}
		lastSeqNum = seqNum

		// Update network statistics
		appState.IncrementRX()

		// Calculate and log network quality metrics
		if packetsReceived%100 == 0 && packetsReceived > 0 {
			lossRate := float32(packetsLost) / float32(packetsReceived)
			logger.Info("Network Quality - Received: %d, Lost: %d (%.2f%%), Seq: %d",
				packetsReceived, packetsLost, lossRate*100, seqNum)

			// Report significant packet loss
			if lossRate > 0.05 { // More than 5% loss
				appState.AddMessage(fmt.Sprintf("High packet loss: %.1f%%", lossRate*100), "warning")
			}
		}

		// Send audio to premium jitter buffer for processing
		audioProcessor.AddToJitterBuffer(seqNum, samples)

		// QUICK FIX: Also send directly to playback channel
		select {
		case incomingAudio <- samples:
			// Successfully queued for playback
		default:
			// Channel full, skip to prevent blocking network thread
			logger.Debug("Playback channel full, dropping frame")
		}

		// Calculate max amplitude for logging (but don't set audio level here - jitter buffer handles that)
		maxAmp := maxAmplitude(samples)
		networkFrameCount++
		if maxAmp > 50 && networkFrameCount%50 == 0 {
			logger.Debug("Receiving audio (seq: %d, amplitude: %d)", seqNum, maxAmp)
		}
	}
}

// Handle incoming chat messages - FIXED PARSING
func handleIncomingChatMessage(data []byte) {
	var chatMsg struct {
		Type      string `json:"type"`
		GUID      string `json:"guid"`
		Channel   string `json:"channel"`
		Username  string `json:"username"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &chatMsg); err != nil {
		logger.Error("Failed to parse incoming chat message: %v", err)
		return
	}

	logger.Debug("Chat message - Channel: %s, User: %s, Message: %s, Timestamp: %s",
		chatMsg.Channel, chatMsg.Username, chatMsg.Message, chatMsg.Timestamp)

	// Create consistent format: [HH:MM] <username> message
	// Use the timestamp from server, but ensure consistent format
	var formattedTimestamp string
	if len(chatMsg.Timestamp) == 5 && chatMsg.Timestamp[2] == ':' {
		// Already HH:MM format
		formattedTimestamp = fmt.Sprintf("[%s]", chatMsg.Timestamp)
	} else {
		// Use current time if server timestamp is weird
		now := time.Now()
		formattedTimestamp = fmt.Sprintf("[%02d:%02d]", now.Hour(), now.Minute())
	}

	// CONSISTENT FORMAT: [HH:MM] <username> message
	chatDisplayMsg := fmt.Sprintf("%s <%s> %s", formattedTimestamp, chatMsg.Username, chatMsg.Message)

	// Add to app state as a chat message - ONLY ONCE
	appState.AddMessage(chatDisplayMsg, "chat")

	logger.Info("Added chat message: %s", chatDisplayMsg)
}

// Handle incoming encrypted chat messages
func handleIncomingEncryptedChatMessage(data []byte) {
	var encryptedMsg struct {
		Type      string `json:"type"`
		GUID      string `json:"guid"`
		Channel   string `json:"channel"`
		Username  string `json:"username"`
		Encrypted bool   `json:"encrypted"`
		Payload   string `json:"payload"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &encryptedMsg); err != nil {
		logger.Error("Failed to parse encrypted chat message: %v", err)
		return
	}

	logger.Debug("Encrypted message from %s in %s", encryptedMsg.Username, encryptedMsg.Channel)

	// Check if we have crypto ready
	if !cryptoReady || !clientCrypto.IsReady() {
		logger.Error("Received encrypted message but crypto not ready")
		return
	}

	// Decode the payload
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedMsg.Payload)
	if err != nil {
		logger.Error("Invalid base64 payload in encrypted message: %v", err)
		return
	}

	// Decrypt the message
	decryptedMessage, err := clientCrypto.DecryptMessage(encryptedData)
	if err != nil {
		logger.Error("Failed to decrypt message: %v", err)
		return
	}

	logger.Debug("Decrypted message: %s", decryptedMessage)

	// Create consistent format: [HH:MM] <username> message
	var formattedTimestamp string
	if len(encryptedMsg.Timestamp) == 5 && encryptedMsg.Timestamp[2] == ':' {
		formattedTimestamp = fmt.Sprintf("[%s]", encryptedMsg.Timestamp)
	} else {
		now := time.Now()
		formattedTimestamp = fmt.Sprintf("[%02d:%02d]", now.Hour(), now.Minute())
	}

	// CONSISTENT FORMAT: [HH:MM] <username> message
	chatDisplayMsg := fmt.Sprintf("%s <%s> %s", formattedTimestamp, encryptedMsg.Username, decryptedMessage)

	// Add to app state as a chat message
	appState.AddMessage(chatDisplayMsg, "chat")

	logger.Info("Added decrypted chat message: %s", chatDisplayMsg)
}

// Handle chat history - FIXED PARSING
func handleChatHistory(data []byte) {
	var historyMsg struct {
		Type     string `json:"type"`
		GUID     string `json:"guid"`
		Channel  string `json:"channel"`
		Messages []struct {
			Username  string    `json:"username"`
			Message   string    `json:"message"`
			Timestamp time.Time `json:"timestamp"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(data, &historyMsg); err != nil {
		logger.Error("Failed to parse chat history: %v", err)
		return
	}

	logger.Info("Received %d chat history messages for channel %s", len(historyMsg.Messages), historyMsg.Channel)

	// Add history messages with consistent formatting
	for _, msg := range historyMsg.Messages {
		// Format timestamp consistently as [HH:MM]
		timestamp := fmt.Sprintf("[%02d:%02d]", msg.Timestamp.Hour(), msg.Timestamp.Minute())

		// CONSISTENT FORMAT: [HH:MM] <username> message
		chatDisplayMsg := fmt.Sprintf("%s <%s> %s", timestamp, msg.Username, msg.Message)

		// Add as chat message
		appState.AddMessage(chatDisplayMsg, "chat")
		logger.Debug("Added history message: %s", chatDisplayMsg)
	}

	if len(historyMsg.Messages) > 0 {
		appState.AddMessage(fmt.Sprintf("--- Loaded %d recent messages for #%s ---", len(historyMsg.Messages), historyMsg.Channel), "info")
	}
}

func startPingLoop(conn *net.UDPConn) {
	logger.Debug("Starting ping loop to maintain connection")

	for {
		ping := map[string]string{"type": "ping"}
		data, _ := json.Marshal(ping)
		conn.Write(data)
		logger.Debug("Sent ping to server")
		time.Sleep(10 * time.Second)
	}
}
