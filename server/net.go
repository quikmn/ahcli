package main

import (
	"ahcli/common"
	"encoding/json"
	"fmt"
	"net"
)

func startUDPServer(config *ServerConfig) {
	addr := net.UDPAddr{
		Port: config.ListenPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Failed to start UDP server:", err)
		return
	}
	defer conn.Close()
	fmt.Printf("Listening on UDP %d...\n", config.ListenPort)

	buffer := make([]byte, 4096)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("UDP read error:", err)
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

			fmt.Printf("Client %s connected from %s\n", nickname, addr.String())

			resp := common.ConnectAccepted{
				Type:     "accept",
				Nickname: nickname,
				MOTD:     config.MOTD,
				Channels: []string{"General", "AFK"}, // TODO: derive from config
				Users:    listNicknames(),
			}
			sendJSON(conn, addr, resp)

		case "change_channel":
			var req struct {
				Type    string `json:"type"`
				Channel string `json:"channel"`
			}
			if err := json.Unmarshal(data, &req); err != nil {
				fmt.Println("Malformed change_channel packet from", addr)
				return
			}

			if !channelExists(req.Channel) {
				fmt.Printf("Client at %s tried to switch to invalid channel: %s\n", addr, req.Channel)
				return
			}

			if updated := updateClientChannel(addr, req.Channel); updated {
				fmt.Printf("Client at %s switched to channel: %s\n", addr, req.Channel)
				ack := map[string]string{
					"type":    "channel_changed",
					"channel": req.Channel,
				}
				sendJSON(conn, addr, ack)
			} else {
				nack := map[string]string{
					"type":    "error",
					"message": "Could not switch channel",
				}
				sendJSON(conn, addr, nack)
			}

		case "ping":
			pong := map[string]string{"type": "pong"}
			sendJSON(conn, addr, pong)
		}
		return
	}

	// Not JSON: treat as raw audio
	client := getClientByAddr(addr)
	if client == nil {
		fmt.Printf("Received audio from unknown client: %s\n", addr)
		return
	}

	// Log and forward raw audio
	fmt.Printf("[AUDIO] %s (%s) sent %d bytes to channel %s\n", client.Nickname, addr, len(data), client.Channel)
	relayCount := 0
	state.Lock()
	for _, other := range state.Clients {
		if other.Channel == client.Channel && other.Addr.String() != addr.String() {
			_, err := conn.WriteToUDP(data, other.Addr)
			if err != nil {
				fmt.Printf("[AUDIO] Relay to %s failed: %v\n", other.Addr, err)
			} else {
				relayCount++
			}
		}
	}
	state.Unlock()

	fmt.Printf("[AUDIO] Relayed to %d peer(s)\n", relayCount)
}

func sendJSON(conn *net.UDPConn, addr *net.UDPAddr, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		fmt.Println("Marshal error:", err)
		return
	}
	conn.WriteToUDP(payload, addr)
}
