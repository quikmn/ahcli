package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"ahcli/common"
)

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
		return fmt.Errorf("no response from server: %v", err)
	}

	var resp map[string]interface{}
	json.Unmarshal(buffer[:n], &resp)

	switch resp["type"] {
	case "accept":
		var accepted common.ConnectAccepted
		json.Unmarshal(buffer[:n], &accepted)
		fmt.Println("Connected as:", accepted.Nickname)
		fmt.Println("MOTD:", accepted.MOTD)
		fmt.Println("Channels:", accepted.Channels)
		fmt.Println("Users:", accepted.Users)
	case "reject":
		var reject common.Reject
		json.Unmarshal(buffer[:n], &reject)
		return fmt.Errorf("rejected: %s", reject.Message)
	default:
		return fmt.Errorf("unexpected response")
	}

	conn.SetReadDeadline(time.Time{})
	serverConn = conn

	go handleUserInput(conn)
	go handleServerResponses(conn)
	go startPingLoop(conn)

	select {}
}

func handleUserInput(conn *net.UDPConn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		inputRaw, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		input := strings.TrimSpace(inputRaw)

		if strings.HasPrefix(strings.ToLower(input), "/join ") {
			channel := strings.TrimSpace(input[6:])
			change := map[string]string{
				"type":    "change_channel",
				"channel": channel,
			}
			data, _ := json.Marshal(change)
			conn.Write(data)
			fmt.Println("Requested channel switch to:", channel)
		} else {
			fmt.Println("Unknown command.")
		}
	}
}

func handleServerResponses(conn *net.UDPConn) {
	buffer := make([]byte, 4096)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Disconnected:", err)
			return
		}

		// Try to parse JSON first
		var msg map[string]interface{}
		if err := json.Unmarshal(buffer[:n], &msg); err == nil {
			switch msg["type"] {
			case "channel_changed":
				fmt.Println("Server: You are now in channel", msg["channel"])
			case "error":
				fmt.Println("Server error:", msg["message"])
			case "pong":
				// silently accepted
			default:
				fmt.Println("Server:", msg)
			}
			continue
		}

		// Not JSON, try raw audio
		if n < 4 {
			fmt.Println("[NET] Dropped malformed packet (too small)")
			continue
		}

		// Validate audio packet prefix
		prefix := binary.LittleEndian.Uint16(buffer[0:2])
		if prefix != 0x5541 { // 'AU' 
			fmt.Printf("[NET] Dropped packet with invalid prefix: 0x%04X\n", prefix)
			continue
		}

		fmt.Printf("[NET] Audio packet received: %d bytes\n", n)

		sampleCount := (n - 2) / 2 // skip 2 byte prefix, 2 bytes per sample
		samples := make([]int16, sampleCount)
		err = binary.Read(bytes.NewReader(buffer[2:n]), binary.LittleEndian, &samples)
		if err != nil {
			fmt.Println("[NET] Failed to decode audio:", err)
			continue
		}

		if len(samples) != framesPerBuffer {
			fmt.Printf("[NET] Dropped frame with wrong length: got %d, expected %d\n", len(samples), framesPerBuffer)
			continue
		}

		// Calculate max amplitude for debugging
		maxAmp := maxAmplitude(samples)
		fmt.Printf("[NET] Decoded %d samples, max amplitude: %d\n", len(samples), maxAmp)

		select {
		case incomingAudio <- samples:
			fmt.Printf("[NET] Queued %d samples to playback buffer\n", len(samples))
		default:
			fmt.Println("[NET] Playback buffer full, dropping packet")
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