package main

import (
	"ahcli/common"
	"encoding/json"
	"fmt"
	"net"
	"time"
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

	req := common.ConnectRequest{
		Type:     "connect",
		Nicklist: config.Nickname,
	}
	data, _ := json.Marshal(req)
	conn.Write(data)

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
	return nil
}
