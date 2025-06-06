// FILE: server/net.go

package main

import (
	"ahcli/common"
	"encoding/json"
	"net"
	"time"
)

func startUDPServer(config *ServerConfig) {
	addr := net.UDPAddr{
		Port: config.ListenPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		LogError("Failed to start UDP server: %v", err)
		return
	}
	defer conn.Close()
	LogInfo("Listening on UDP %d...", config.ListenPort)

	buffer := make([]byte, 4096)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			LogError("UDP read error: %v", err)
			continue
		}

		// Copy data so it's safe across goroutines
		packet := make([]byte, n)
		copy(packet, buffer[:n])
		go handlePacket(conn, packet, clientAddr, config)
	}
}

func handlePacket(conn *net.UDPConn, data []byte, addr *net.UDPAddr, config *ServerConfig) {
	// Try JSON parsing first
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		switch raw["type"] {
		case "connect":
			handleConnect(conn, data, addr, config)

		case "change_channel":
			handleChangeChannel(conn, data, addr)

		case "chat":
			handleChatMessage(conn, data, addr)

		case "ping":
			handlePing(conn, addr)
		}
		return
	}

	// Not JSON: treat as raw audio
	handleAudioData(conn, data, addr)
}

func handleConnect(conn *net.UDPConn, data []byte, addr *net.UDPAddr, config *ServerConfig) {
	var req common.ConnectRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return
	}

	var nickname string
	for _, try := range req.Nicklist {
		if reserveNickname(try, addr) {
			nickname = try
			break
		}
	}
	if nickname == "" {
		reject := common.Reject{Type: "reject", Message: "All nicknames are taken"}
		sendJSON(conn, addr, reject)
		return
	}

	LogClient("Client %s connected from %s", nickname, addr.String())

	// Get channel names from config
	channelNames := make([]string, len(config.Channels))
	for i, ch := range config.Channels {
		channelNames[i] = ch.Name
	}

	resp := common.ConnectAccepted{
		Type:       "accept",
		Nickname:   nickname,
		ServerName: config.ServerName,
		MOTD:       config.MOTD,
		Channels:   channelNames,
		Users:      listNicknames(),
	}
	sendJSON(conn, addr, resp)

	// Send recent chat history for the default channel (General)
	if chatStorage != nil && chatStorage.enabled {
		defaultChannelGUID := GetChannelGUID("General")
		if defaultChannelGUID != "" {
			sendRecentChatHistory(conn, addr, defaultChannelGUID)
		}
	}
}

func handleChangeChannel(conn *net.UDPConn, data []byte, addr *net.UDPAddr) {
	var req struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		LogError("Malformed change_channel packet from %s", addr)
		return
	}

	if !channelExists(req.Channel) {
		LogClient("Client at %s tried to switch to invalid channel: %s", addr, req.Channel)
		return
	}

	if updated := updateClientChannel(addr, req.Channel); updated {
		LogClient("Client at %s switched to channel: %s", addr, req.Channel)
		ack := map[string]string{
			"type":    "channel_changed",
			"channel": req.Channel,
		}
		sendJSON(conn, addr, ack)
		broadcastChannelUserUpdate(conn)

		// Send recent chat history for the new channel
		if chatStorage != nil && chatStorage.enabled {
			channelGUID := GetChannelGUID(req.Channel)
			if channelGUID != "" {
				sendRecentChatHistory(conn, addr, channelGUID)
			}
		}
	} else {
		nack := map[string]string{
			"type":    "error",
			"message": "Could not switch channel",
		}
		sendJSON(conn, addr, nack)
	}
}

func handleChatMessage(conn *net.UDPConn, data []byte, addr *net.UDPAddr) {
	var chatMsg struct {
		Type     string `json:"type"`
		Channel  string `json:"channel"`  // Channel name for routing
		Message  string `json:"message"`  // The actual message
		Username string `json:"username"` // Who sent it
	}

	if err := json.Unmarshal(data, &chatMsg); err != nil {
		LogError("Malformed chat message from %s: %v", addr, err)
		return
	}

	// Get the client who sent this
	client := getClientByAddr(addr)
	if client == nil {
		LogError("Chat message from unknown client: %s", addr)
		return
	}

	// Validate message content
	if chatMsg.Message == "" {
		LogDebug("Empty chat message from %s, ignoring", client.Nickname)
		return
	}

	// Get channel GUID for routing
	channelGUID := GetChannelGUID(client.Channel)
	if channelGUID == "" {
		LogError("No GUID found for channel %s", client.Channel)
		return
	}

	// Store the message in chat storage
	if chatStorage != nil && chatStorage.enabled {
		err := chatStorage.StoreMessage(channelGUID, client.Channel, client.Nickname, chatMsg.Message)
		if err != nil {
			LogError("Failed to store chat message: %v", err)
			// Continue anyway - still broadcast the message
		}
	}

	LogClient("Chat in %s (%s): <%s> %s", client.Channel, channelGUID, client.Nickname, chatMsg.Message)

	// Broadcast to all users in the same channel
	broadcastChatMessage(conn, channelGUID, client.Channel, client.Nickname, chatMsg.Message)
}

func handlePing(conn *net.UDPConn, addr *net.UDPAddr) {
	pong := map[string]string{"type": "pong"}
	sendJSON(conn, addr, pong)
}

func handleAudioData(conn *net.UDPConn, data []byte, addr *net.UDPAddr) {
	client := getClientByAddr(addr)
	if client == nil {
		LogAudio("Received audio from unknown client: %s", addr)
		return
	}

	// Log and forward raw audio
	LogAudio("%s (%s) sent %d bytes to channel %s", client.Nickname, addr, len(data), client.Channel)
	relayCount := 0
	state.Lock()
	for _, other := range state.Clients {
		if other.Channel == client.Channel && other.Addr.String() != addr.String() {
			_, err := conn.WriteToUDP(data, other.Addr)
			if err != nil {
				LogAudio("Relay to %s failed: %v", other.Addr, err)
			} else {
				relayCount++
			}
		}
	}
	state.Unlock()

	LogAudio("Relayed to %d peer(s)", relayCount)
}

func broadcastChatMessage(conn *net.UDPConn, channelGUID, channelName, username, message string) {
	// Create chat message for broadcast
	chatBroadcast := map[string]interface{}{
		"type":      "chat_message",
		"guid":      channelGUID,
		"channel":   channelName,
		"username":  username,
		"message":   message,
		"timestamp": time.Now().Format("15:04:05"), // HH:MM:SS format
	}

	// Get all clients in the same channel
	var clientAddrs []*net.UDPAddr
	state.Lock()
	for _, client := range state.Clients {
		if client.Channel == channelName {
			clientAddrs = append(clientAddrs, client.Addr)
		}
	}
	state.Unlock()

	// Broadcast to all clients in the channel
	broadcastCount := 0
	for _, clientAddr := range clientAddrs {
		err := sendJSON(conn, clientAddr, chatBroadcast)
		if err != nil {
			LogError("Failed to broadcast chat to %s: %v", clientAddr, err)
		} else {
			broadcastCount++
		}
	}

	LogDebug("Broadcasted chat message to %d clients in %s", broadcastCount, channelName)
}

func sendRecentChatHistory(conn *net.UDPConn, addr *net.UDPAddr, channelGUID string) {
	if chatStorage == nil || !chatStorage.enabled {
		return
	}

	// Get recent messages for this channel
	recentMessages := chatStorage.GetRecentMessages(channelGUID, chatStorage.recentOnJoin)
	if len(recentMessages) == 0 {
		LogDebug("No recent chat history for channel GUID %s", channelGUID)
		return
	}

	// Send chat history as a batch
	historyMsg := map[string]interface{}{
		"type":     "chat_history",
		"guid":     channelGUID,
		"channel":  GetChannelName(channelGUID),
		"messages": recentMessages,
	}

	err := sendJSON(conn, addr, historyMsg)
	if err != nil {
		LogError("Failed to send chat history to %s: %v", addr, err)
	} else {
		LogDebug("Sent %d recent chat messages to %s", len(recentMessages), addr)
	}
}

func sendJSON(conn *net.UDPConn, addr *net.UDPAddr, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		LogError("Marshal error: %v", err)
		return err
	}
	_, err = conn.WriteToUDP(payload, addr)
	return err
}

func broadcastChannelUserUpdate(conn *net.UDPConn) {
	// Build current channel user mapping
	channelUsers := make(map[string][]string)

	state.Lock()
	// Initialize all channels with empty arrays
	for _, client := range state.Clients {
		if _, exists := channelUsers[client.Channel]; !exists {
			channelUsers[client.Channel] = make([]string, 0)
		}
	}
	// Populate with actual users
	for _, client := range state.Clients {
		channelUsers[client.Channel] = append(channelUsers[client.Channel], client.Nickname)
	}

	// Get all client addresses
	clientAddrs := make([]*net.UDPAddr, 0, len(state.Clients))
	for _, client := range state.Clients {
		clientAddrs = append(clientAddrs, client.Addr)
	}
	state.Unlock()

	// Broadcast to all clients
	update := map[string]interface{}{
		"type":         "channel_users_update",
		"channelUsers": channelUsers,
	}

	for _, addr := range clientAddrs {
		sendJSON(conn, addr, update)
	}
}
