package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var serverConfig *ServerConfig

type Channel struct {
	Name        string `json:"name"`
	AllowSpeak  bool   `json:"allow_speak"`
	AllowListen bool   `json:"allow_listen"`
}

type ServerConfig struct {
	ServerName string    `json:"server_name"`
	ListenPort int       `json:"listen_port"`
	SharedKey  string    `json:"shared_key"`
	AdminKey   string    `json:"admin_key"`
	MOTD       string    `json:"motd"`
	Channels   []Channel `json:"channels"`
}

func loadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {
	config, err := loadServerConfig("config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	serverConfig = config // <- This is the fix

	fmt.Println("Server config loaded:")
	fmt.Printf("Server Name: %s\n", config.ServerName)
	fmt.Printf("Port: %d\n", config.ListenPort)
	fmt.Printf("MOTD: %s\n", config.MOTD)
	fmt.Println("Channels:")
	for _, ch := range config.Channels {
		fmt.Printf(" - %s (speak: %t, listen: %t)\n", ch.Name, ch.AllowSpeak, ch.AllowListen)
	}

	// This line is crucial!
	startUDPServer(config)
}

