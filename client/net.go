package main

import (
	"bufio"
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
		if err := startPlayback(); err != nil {
    		fmt.Println("Playback init failed:", err)
		}
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

	// Disable read timeout for long-running receive loop
	conn.SetReadDeadline(time.Time{})
	serverConn = conn

	// Start background listeners
	go handleUserInput(conn)
	go handleServerResponses(conn)
	go startPingLoop(conn)

	// Block forever
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

		var msg map[string]interface{}
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			fmt.Println("[NET] Invalid server message:", err)
			continue
		}

		switch msg["type"] {
		case "channel_changed":
			fmt.Println("Server: You are now in channel", msg["channel"])

		case "error":
			fmt.Println("Server error:", msg["message"])

		case "pong":
			// silently accepted

		case "audio":
			var audioMsg struct {
				Type string  `json:"type"`
				Data []int16 `json:"data"`
			}
			if err := json.Unmarshal(buffer[:n], &audioMsg); err != nil {
				fmt.Println("[NET] Failed to parse audio packet:", err)
				continue
			}

			fmt.Printf("[NET] Received audio packet: %d samples\n", len(audioMsg.Data))

			if len(audioMsg.Data) != framesPerBuffer {
				fmt.Printf("[NET] Dropping frame with invalid length: %d\n", len(audioMsg.Data))
				continue
			}

			select {
			case incomingAudio <- audioMsg.Data:
			default:
				fmt.Println("[NET] Audio buffer full, dropping packet")
			}

		default:
			fmt.Println("Server:", msg)
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
