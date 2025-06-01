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
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.SetConnected(true, accepted.Nickname, accepted.ServerName, accepted.MOTD)
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.SetChannel(currentChannel)
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
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
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
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
	
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			LogError("Disconnected: %v", err)
			
			// PURE APPSTATE: Only update AppState - observer handles WebTUI
			appState.SetConnected(false, "", "", "")
			appState.AddMessage("Disconnected from server", "error")
			return
		}

		// Try to parse JSON first
		var msg map[string]interface{}
		if err := json.Unmarshal(buffer[:n], &msg); err == nil {
			switch msg["type"] {
			case "channel_changed":
				channelName := msg["channel"].(string)
				currentChannel = channelName
				
				// PURE APPSTATE: Only update AppState - observer handles WebTUI
				appState.SetChannel(channelName)
				LogInfo("You are now in channel: %s", channelName)
				
			case "error":
				errorMsg := msg["message"].(string)
				
				// PURE APPSTATE: Only update AppState - observer handles WebTUI
				appState.AddMessage(fmt.Sprintf("Server error: %s", errorMsg), "error")
				LogError("Server error: %s", errorMsg)
				
			case "pong":
				// silently accepted
			default:
				LogInfo("Server message: %v", msg)
			}
			continue
		}

		// Not JSON, try raw audio
		if n < 4 {
			LogError("Dropped malformed packet (too small)")
			continue
		}

		// Validate audio packet prefix
		prefix := binary.LittleEndian.Uint16(buffer[0:2])
		if prefix != 0x5541 { // 'AU' 
			LogError("Dropped packet with invalid prefix: 0x%04X", prefix)
			continue
		}

		sampleCount := (n - 2) / 2 // skip 2 byte prefix, 2 bytes per sample
		samples := make([]int16, sampleCount)
		err = binary.Read(bytes.NewReader(buffer[2:n]), binary.LittleEndian, &samples)
		if err != nil {
			LogError("Failed to decode audio: %v", err)
			continue
		}

		if len(samples) != framesPerBuffer {
			LogError("Dropped frame with wrong length: got %d, expected %d", len(samples), framesPerBuffer)
			continue
		}

		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.IncrementRX()
		
		maxAmp := maxAmplitude(samples)
		if maxAmp > 50 {
			// PURE APPSTATE: Only update AppState - observer handles WebTUI
			level := int(float64(maxAmp) / 32767.0 * 100)
			appState.SetAudioLevel(level)
		}

		// Calculate max amplitude - only log every 50 frames (once per second)
		networkFrameCount++
		if maxAmp > 50 && networkFrameCount%50 == 0 {
			LogInfo("Receiving audio (amplitude: %d)", maxAmp)
		}

		select {
		case incomingAudio <- samples:
			// Successfully queued
		default:
			LogError("Playback buffer full, dropping packet")
			
			// PURE APPSTATE: Only update AppState - observer handles WebTUI
			appState.AddMessage("Audio buffer overflow", "error")
		}
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