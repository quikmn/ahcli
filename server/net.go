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

		go handlePacket(conn, buffer[:n], clientAddr, config)
	}
}

func handlePacket(conn *net.UDPConn, data []byte, addr *net.UDPAddr, config *ServerConfig) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		fmt.Println("Malformed packet from", addr)
		return
	}

	switch raw["type"] {
	case "connect":
		var req common.ConnectRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return
		}
		nickname := pickAvailableNickname(req.Nicklist)
		if nickname == "" {
			reject := common.Reject{Type: "reject", Message: "All nicknames are taken"}
			sendJSON(conn, addr, reject)
			return
		}

		// Register client in state (mock for now)
		fmt.Printf("Client %s connected from %s\n", nickname, addr.String())


		resp := common.ConnectAccepted{
			Type:     "accept",
			Nickname: nickname,
			MOTD:     config.MOTD,
			Channels: []string{"General", "AFK"}, // TODO: derive from config
			Users:    []string{"quikmn"},         // TODO: real user state
		}
		sendJSON(conn, addr, resp)
	}
}

func sendJSON(conn *net.UDPConn, addr *net.UDPAddr, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		fmt.Println("Marshal error:", err)
		return
	}
	conn.WriteToUDP(payload, addr)
}

func pickAvailableNickname(nicks []string) string {
	// TODO: use real state later
	for _, nick := range nicks {
		if nick != "taken" { // simulate one being taken
			return nick
		}
	}
	return ""
}
