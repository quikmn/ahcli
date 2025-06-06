// FILE: server/main.go

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Channel struct {
	GUID        string `json:"guid"`         // NEW: Permanent channel identifier
	Name        string `json:"name"`         // Human-readable name (changeable)
	AllowSpeak  bool   `json:"allow_speak"`  // Can users transmit voice
	AllowListen bool   `json:"allow_listen"` // Can users receive voice
}

type ChatConfig struct {
	Enabled          bool   `json:"enabled"`             // Enable/disable chat system
	LogFile          string `json:"log_file"`            // Chat log file path
	MaxMessages      int    `json:"max_messages"`        // Circular buffer size
	LoadRecentOnJoin int    `json:"load_recent_on_join"` // Messages to load when joining channel
}

type ServerConfig struct {
	ServerName string     `json:"server_name"`
	ListenPort int        `json:"listen_port"`
	SharedKey  string     `json:"shared_key"`
	AdminKey   string     `json:"admin_key"`
	MOTD       string     `json:"motd"`
	Channels   []Channel  `json:"channels"`
	Chat       ChatConfig `json:"chat"` // NEW: Chat configuration
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
	LogDebug("Chat enabled: %t", config.Chat.Enabled)

	for _, ch := range config.Channels {
		LogDebug("Channel: %s (GUID: %s, speak: %t, listen: %t)",
			ch.Name, ch.GUID, ch.AllowSpeak, ch.AllowListen)
	}

	// Initialize chat storage system
	err = InitChatStorage(config)
	if err != nil {
		fmt.Printf("Failed to initialize chat system: %v\n", err)
		LogError("Failed to initialize chat system: %v", err)
		return
	}
	defer CloseChatStorage()

	LogInfo("Starting UDP server on port %d", config.ListenPort)
	startUDPServer(config)
}
