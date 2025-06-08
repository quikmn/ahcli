// FILE: server/main.go

package main

import (
	"ahcli/common/logger"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Channel struct {
	GUID        string `json:"guid"`         // Permanent channel identifier
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
	Chat       ChatConfig `json:"chat"`
}

var (
	serverConfig *ServerConfig
	debugMode    = flag.Bool("debug", false, "Enable debug logging")
)

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
	// Parse command line flags FIRST
	flag.Parse()

	// Initialize unified logging system
	err := logger.Init("server")
	if err != nil {
		fmt.Printf("Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Set debug mode from command line flag
	logger.SetDebugMode(*debugMode)

	logger.Info("=== AHCLI Server Starting ===")
	if *debugMode {
		logger.Debug("Debug mode enabled")
	}
	logger.Info("Log file: %s", logger.GetLogPath())

	// Load configuration
	config, err := loadServerConfig("config.json")
	if err != nil {
		logger.Fatal("Failed to load config: %v", err)
		return
	}

	serverConfig = config
	logger.Info("Server config loaded successfully")
	logger.Debug("Server Name: %s", config.ServerName)
	logger.Debug("Port: %d", config.ListenPort)
	logger.Debug("MOTD: %s", config.MOTD)
	logger.Debug("Chat enabled: %t", config.Chat.Enabled)

	for _, ch := range config.Channels {
		logger.Debug("Channel: %s (GUID: %s, speak: %t, listen: %t)",
			ch.Name, ch.GUID, ch.AllowSpeak, ch.AllowListen)
	}

	// Initialize chat storage system
	err = InitChatStorage(config)
	if err != nil {
		logger.Fatal("Failed to initialize chat system: %v", err)
		return
	}
	defer CloseChatStorage()
	logger.Info("Chat system initialized - log: %s, max messages: %d",
		config.Chat.LogFile, config.Chat.MaxMessages)

	// Initialize server crypto system
	err = InitServerCrypto()
	if err != nil {
		logger.Fatal("Failed to initialize crypto system: %v", err)
		return
	}
	logger.Info("Server crypto system initialized")

	logger.Info("Starting UDP server on port %d", config.ListenPort)
	startUDPServer(config)
}
