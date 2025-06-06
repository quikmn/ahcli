// FILE: client/net.go

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"ahcli/common"
)

var currentChannel string

func connectToServer(config *ClientConfig) error {
	target := config.Servers[config.PreferredServer].IP
	raddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send connect request
	req := common.ConnectRequest{
		Type:     "connect",
		Nicklist: config.Nickname,
	}
	data, _ := json.Marshal(req)
	conn.Write(data)

	// Wait for response
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
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

		LogInfo("Connected as: %s", accepted.Nickname)
		LogInfo("MOTD: %s", accepted.MOTD)
		LogInfo("Channels: %v", accepted.Channels)
		LogInfo("Users: %v", accepted.Users)

	case "reject":
		var reject common.Reject
		json.Unmarshal(buffer[:n], &reject)
		return err
	default:
		return err
	}

	conn.SetReadDeadline(time.Time{})
	serverConn = conn

	go handleServerResponses(conn)
	go startPingLoop(conn)

	select {}
}

// Called from Web UI
func changeChannel(channel string) {
	if serverConn == nil {
		LogError("Not connected to server")
		return
	}

	change := map[string]string{
		"type":    "change_channel",
		"channel": channel,
	}
	data, _ := json.Marshal(change)
	serverConn.Write(data)

	LogInfo("Requested channel switch to: %s", channel)
}

// Send chat message to server
func sendChatMessage(message string) {
	if serverConn == nil {
		LogError("Not connected to server")
		appState.AddMessage("Cannot send chat: not connected", "error")
		return
	}

	if currentChannel == "" {
		LogError("No current channel for chat")
		appState.AddMessage("Cannot send chat: no channel", "error")
		return
	}

	// Get current user nickname
	state := appState.GetState()
	nickname := state["nickname"].(string)

	chatMsg := map[string]string{
		"type":     "chat",
		"channel":  currentChannel,
		"message":  message,
		"username": nickname,
	}

	data, err := json.Marshal(chatMsg)
	if err != nil {
		LogError("Failed to marshal chat message: %v", err)
		return
	}

	_, err = serverConn.Write(data)
	if err != nil {
		LogError("Failed to send chat message: %v", err)
		appState.AddMessage("Failed to send chat message", "error")
	} else {
		LogInfo("✅ Sent chat message to server: %s", message)
	}
}

func handleServerResponses(conn *net.UDPConn) {
	buffer := make([]byte, 4096)
	var networkFrameCount int
	var lastSeqNum uint16 = 0
	var packetsReceived int
	var packetsLost int

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			LogError("Disconnected: %v", err)
			appState.SetConnected(false, "", "", "")
			appState.AddMessage("Disconnected from server", "error")
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
				LogInfo("You are now in channel: %s", channelName)

			case "error":
				errorMsg := msg["message"].(string)
				appState.AddMessage(fmt.Sprintf("Server error: %s", errorMsg), "error")
				LogError("Server error: %s", errorMsg)

			case "pong":
				// silently accepted

			case "channel_users_update":
				var update struct {
					ChannelUsers map[string][]string `json:"channelUsers"`
				}
				if err := json.Unmarshal(buffer[:n], &update); err == nil {
					appState.SetChannelUsers(update.ChannelUsers)
				}

			case "chat_message":
				LogInfo("🔍 DEBUG: Received chat_message from server")
				handleIncomingChatMessage(buffer[:n])

			case "chat_history":
				LogInfo("🔍 DEBUG: Received chat_history from server")
				handleChatHistory(buffer[:n])

			default:
				LogInfo("Server message: %v", msg)
			}
			continue
		}

		// Not JSON, try premium audio packet
		if n < 6 { // Minimum: 2 bytes prefix + 2 bytes seq + 2 bytes audio
			LogError("Dropped malformed packet (too small): %d bytes", n)
			continue
		}

		// Validate audio packet prefix
		prefix := binary.LittleEndian.Uint16(buffer[0:2])
		if prefix != 0x5541 { // 'AU'
			LogError("Dropped packet with invalid prefix: 0x%04X", prefix)
			continue
		}

		// Extract sequence number (premium packets)
		seqNum := binary.LittleEndian.Uint16(buffer[2:4])

		// Calculate audio payload size
		sampleCount := (n - 4) / 2 // Skip 4 bytes (prefix + seq), 2 bytes per sample
		if sampleCount != framesPerBuffer {
			LogError("Dropped frame with wrong length: got %d samples, expected %d", sampleCount, framesPerBuffer)
			continue
		}

		// Decode audio samples
		samples := make([]int16, sampleCount)
		err = binary.Read(bytes.NewReader(buffer[4:n]), binary.LittleEndian, &samples)
		if err != nil {
			LogError("Failed to decode audio: %v", err)
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
					LogDebug("Packet loss detected: expected %d, got %d (%d packets lost)",
						expectedSeq, seqNum, lost)
				} else {
					// Out of order packet (late arrival)
					LogDebug("Out-of-order packet: expected %d, got %d", expectedSeq, seqNum)
				}
			}
		}
		lastSeqNum = seqNum

		// Update network statistics
		appState.IncrementRX()

		// Calculate and log network quality metrics
		if packetsReceived%100 == 0 && packetsReceived > 0 {
			lossRate := float32(packetsLost) / float32(packetsReceived)
			LogInfo("Network Quality - Received: %d, Lost: %d (%.2f%%), Seq: %d",
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
			LogDebug("Playback channel full, dropping frame")
		}

		// Calculate max amplitude for logging (but don't set audio level here - jitter buffer handles that)
		maxAmp := maxAmplitude(samples)
		networkFrameCount++
		if maxAmp > 50 && networkFrameCount%50 == 0 {
			LogInfo("Receiving premium audio (seq: %d, amplitude: %d)", seqNum, maxAmp)
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
		LogError("Failed to parse incoming chat message: %v", err)
		return
	}

	LogInfo("🔍 DEBUG: Raw server data - Channel: %s, User: %s, Message: %s, Timestamp: %s",
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

	LogInfo("🔍 DEBUG: Added formatted chat message: %s", chatDisplayMsg)
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
		LogError("Failed to parse chat history: %v", err)
		return
	}

	LogInfo("🔍 DEBUG: Received %d chat history messages for channel %s", len(historyMsg.Messages), historyMsg.Channel)

	// Add history messages with consistent formatting
	for _, msg := range historyMsg.Messages {
		// Format timestamp consistently as [HH:MM]
		timestamp := fmt.Sprintf("[%02d:%02d]", msg.Timestamp.Hour(), msg.Timestamp.Minute())

		// CONSISTENT FORMAT: [HH:MM] <username> message
		chatDisplayMsg := fmt.Sprintf("%s <%s> %s", timestamp, msg.Username, msg.Message)

		// Add as chat message
		appState.AddMessage(chatDisplayMsg, "chat")
		LogInfo("🔍 DEBUG: Added history message: %s", chatDisplayMsg)
	}

	if len(historyMsg.Messages) > 0 {
		appState.AddMessage(fmt.Sprintf("--- Loaded %d recent messages for #%s ---", len(historyMsg.Messages), historyMsg.Channel), "info")
	}
}

func startPingLoop(conn *net.UDPConn) {
	for {
		ping := map[string]string{"type": "ping"}
		data, _ := json.Marshal(ping)
		conn.Write(data)
		time.Sleep(10 * time.Second)
	}
}
