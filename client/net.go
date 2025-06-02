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

		// Try to parse JSON first (control messages)
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

		// Note: We no longer directly queue to incomingAudio channel
		// The premium audio processor handles jitter buffering and playback timing
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