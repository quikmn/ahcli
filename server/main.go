package main

import (
	"encoding/json"
	"fmt"
	"os"
)

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

var serverConfig *ServerConfig

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
	// Initialize logging first (parses --debug flag)
	InitLogger()
	defer CloseLogger()

	config, err := loadServerConfig("config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		LogError("Error loading config: %v", err)
		return
	}

	serverConfig = config

	LogInfo("Server config loaded successfully")
	LogDebug("Server Name: %s", config.ServerName)
	LogDebug("Port: %d", config.ListenPort)
	LogDebug("MOTD: %s", config.MOTD)
	for _, ch := range config.Channels {
		LogDebug("Channel: %s (speak: %t, listen: %t)", ch.Name, ch.AllowSpeak, ch.AllowListen)
	}

	LogInfo("Starting UDP server on port %d", config.ListenPort)
	startUDPServer(config)
}