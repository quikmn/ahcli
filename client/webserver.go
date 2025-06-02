// FILE: client/webserver.go

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"io/fs"
	"net/http"
	"sync"
	"time"
	"strconv"

	"github.com/gorilla/websocket"
)

//go:embed web/*
var webFiles embed.FS

type WebTUIState struct {
	sync.RWMutex
	Connected      bool              `json:"connected"`
	Nickname       string            `json:"nickname"`
	ServerName     string            `json:"serverName"`
	CurrentChannel string            `json:"currentChannel"`
	Channels       []string          `json:"channels"`
	ChannelUsers   map[string][]string `json:"channelUsers"`
	PTTActive      bool              `json:"pttActive"`
	AudioLevel     int               `json:"audioLevel"`
	PacketsRx      int               `json:"packetsRx"`
	PacketsTx      int               `json:"packetsTx"`
	ConnectionTime time.Time         `json:"connectionTime"`
	Messages       []WebMessage      `json:"messages"`
	PTTKey         string            `json:"pttKey"`
	
	// NEW: Audio processing state
	AudioPreset    string            `json:"audioPreset"`
	InputLevel     float32           `json:"inputLevel"`
	OutputLevel    float32           `json:"outputLevel"`
}

type WebMessage struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Type      string `json:"type"` // "info", "error", "success", "ptt"
}

var (
	webTUI = &WebTUIState{
		ChannelUsers: make(map[string][]string),
		Messages:     make([]WebMessage, 0),
		PTTKey:       "LSHIFT",
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	wsClients = make(map[*websocket.Conn]bool)
	wsMutex   sync.Mutex
	observersSetup = false
	
	// Global config reference for audio controls
	currentConfig *ClientConfig
)

func StartWebServer() (int, error) {
	// Find available port
	port := findAvailablePort(8080)
	
	// Serve embedded files with proper routing
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		LogError("Failed to create web filesystem: %v", err)
		return 0, err
	}
	http.Handle("/", http.FileServer(http.FS(webFS)))
	
	// API endpoints
	http.HandleFunc("/api/state", handleAPIState)
	http.HandleFunc("/api/command", handleAPICommand)
	http.HandleFunc("/ws", handleWebSocket)

	LogInfo("Starting web server on port %d", port)
	
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			LogError("Web server failed: %v", err)
		}
	}()
	
	// Set up AppState observers - WebTUI becomes pure observer!
	setupAppStateObservers()
	
	return port, nil
}

// setupAppStateObservers makes WebTUI a pure observer of AppState changes
func setupAppStateObservers() {
	if observersSetup {
		return // Only setup once
	}
	
	LogInfo("Setting up WebTUI as AppState observer...")
	
	appState.AddObserver(func(change StateChange) {
		switch change.Type {
		case "ptt":
			if active, ok := change.Data.(bool); ok {
				LogDebug("Observer: PTT state changed to %t", active)
				webTUI.Lock()
				webTUI.PTTActive = active
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "audio_level":
			if level, ok := change.Data.(int); ok {
				webTUI.Lock()
				webTUI.AudioLevel = level
				webTUI.Unlock()
				// Don't broadcast every audio level change - too spammy
			}
			
		case "connection":
			if data, ok := change.Data.(map[string]interface{}); ok {
				LogDebug("Observer: Connection state changed")
				webTUI.Lock()
				if connected, ok := data["connected"].(bool); ok {
					webTUI.Connected = connected
				}
				if nickname, ok := data["nickname"].(string); ok {
					webTUI.Nickname = nickname
				}
				if serverName, ok := data["serverName"].(string); ok {
					webTUI.ServerName = serverName
				}
				if webTUI.Connected {
					webTUI.ConnectionTime = time.Now()
				}
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "channel":
			if channel, ok := change.Data.(string); ok {
				LogDebug("Observer: Channel changed to %s", channel)
				webTUI.Lock()
				webTUI.CurrentChannel = channel
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "channels":
			if channels, ok := change.Data.([]string); ok {
				LogDebug("Observer: Channels list updated")
				webTUI.Lock()
				webTUI.Channels = channels
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "channel_users":
			if channelUsers, ok := change.Data.(map[string][]string); ok {
				LogDebug("Observer: Channel users updated")
				webTUI.Lock()
				webTUI.ChannelUsers = channelUsers
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "message":
			if msg, ok := change.Data.(AppMessage); ok {
				LogDebug("Observer: New message - %s", msg.Message)
				webTUI.Lock()
				webMsg := WebMessage{
					Timestamp: msg.Timestamp,
					Message:   msg.Message,
					Type:      msg.Type,
				}
				webTUI.Messages = append(webTUI.Messages, webMsg)
				
				// Keep only last 100 messages
				if len(webTUI.Messages) > 100 {
					webTUI.Messages = webTUI.Messages[len(webTUI.Messages)-100:]
				}
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "ptt_key":
			if keyName, ok := change.Data.(string); ok {
				LogDebug("Observer: PTT key changed to %s", keyName)
				webTUI.Lock()
				webTUI.PTTKey = keyName
				webTUI.Unlock()
				broadcastUpdate()
			}
			
		case "packets_rx":
			if packets, ok := change.Data.(int); ok {
				webTUI.Lock()
				webTUI.PacketsRx = packets
				webTUI.Unlock()
				// Only broadcast batched updates (every 10 packets)
				broadcastUpdate()
			}
			
		case "packets_tx":
			if packets, ok := change.Data.(int); ok {
				webTUI.Lock()
				webTUI.PacketsTx = packets
				webTUI.Unlock()
				// Only broadcast batched updates (every 10 packets)
				broadcastUpdate()
			}
		}
	})
	
	observersSetup = true
	LogInfo("WebTUI observers setup complete - now pure observer of AppState!")
}

func findAvailablePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port
		}
	}
	return startPort // fallback
}

func handleAPIState(w http.ResponseWriter, r *http.Request) {
	webTUI.RLock()
	defer webTUI.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webTUI)
}

func handleAPICommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	
	var cmd struct {
		Command string `json:"command"`
		Args    string `json:"args"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	
	switch cmd.Command {
	case "join":
		changeChannel(cmd.Args)
		
		// DUAL-WRITE: Update both systems with join message
		appState.AddMessage(fmt.Sprintf("Joining channel: %s", cmd.Args), "info")
		WebTUIAddMessage(fmt.Sprintf("Joining channel: %s", cmd.Args), "info")
		
	case "quit":
		// DUAL-WRITE: Update both systems with quit message
		appState.AddMessage("Disconnecting...", "info")
		WebTUIAddMessage("Disconnecting...", "info")
		// Could trigger graceful shutdown here
		
	case "audio_preset":
		handleAudioPreset(cmd.Args)
		
	case "audio_setting":
		handleAudioSetting(cmd.Args)
		
	case "test_microphone":
		handleTestMicrophone()
		
	case "save_custom_preset":
		handleSaveCustomPreset()
		
	default:
		// DUAL-WRITE: Update both systems with unknown command message
		appState.AddMessage(fmt.Sprintf("Unknown command: %s", cmd.Command), "error")
		WebTUIAddMessage(fmt.Sprintf("Unknown command: %s", cmd.Command), "error")
	}
	
	w.WriteHeader(200)
}

// Audio preset handler
func handleAudioPreset(preset string) {
	LogInfo("Changing audio preset to: %s", preset)
	
	if currentConfig == nil {
		LogError("No config loaded for audio preset change")
		appState.AddMessage("Error: Configuration not loaded", "error")
		return
	}
	
	// Apply preset to config
	applyAudioPreset(currentConfig, preset)
	
	// Apply to processor immediately
	applyAudioConfigToProcessor(currentConfig)
	
	// Update UI state
	webTUI.Lock()
	webTUI.AudioPreset = preset
	webTUI.Unlock()
	
	// Save config to file
	if err := saveClientConfig("settings.config", currentConfig); err != nil {
		LogError("Failed to save audio preset: %v", err)
		appState.AddMessage("Failed to save audio settings", "error")
	} else {
		LogInfo("Audio preset '%s' applied and saved", preset)
		appState.AddMessage(fmt.Sprintf("Audio preset changed to: %s", preset), "success")
	}
	
	broadcastUpdate()
}

// Individual audio setting handler
func handleAudioSetting(argsJSON string) {
	var setting struct {
		Section string      `json:"section"`
		Param   string      `json:"param"`
		Value   interface{} `json:"value"`
	}
	
	if err := json.Unmarshal([]byte(argsJSON), &setting); err != nil {
		LogError("Invalid audio setting JSON: %v", err)
		return
	}
	
	LogInfo("Updating audio setting: %s.%s = %v", setting.Section, setting.Param, setting.Value)
	
	if currentConfig == nil {
		LogError("No config loaded for audio setting change")
		return
	}
	
	// Update config based on section and parameter
	switch setting.Section {
	case "noiseGate":
		switch setting.Param {
		case "enabled":
			if enabled, ok := setting.Value.(bool); ok {
				currentConfig.AudioProcessing.NoiseGate.Enabled = enabled
			}
		case "threshold":
			if threshold, ok := setting.Value.(string); ok {
				if val, err := strconv.ParseFloat(threshold, 32); err == nil {
					currentConfig.AudioProcessing.NoiseGate.ThresholdDB = float32(val)
				}
			}
		}
		
	case "compressor":
		switch setting.Param {
		case "enabled":
			if enabled, ok := setting.Value.(bool); ok {
				currentConfig.AudioProcessing.Compressor.Enabled = enabled
			}
		case "threshold":
			if threshold, ok := setting.Value.(string); ok {
				if val, err := strconv.ParseFloat(threshold, 32); err == nil {
					currentConfig.AudioProcessing.Compressor.ThresholdDB = float32(val)
				}
			}
		case "ratio":
			if ratio, ok := setting.Value.(string); ok {
				if val, err := strconv.ParseFloat(ratio, 32); err == nil {
					currentConfig.AudioProcessing.Compressor.Ratio = float32(val)
				}
			}
		}
		
	case "makeupGain":
		switch setting.Param {
		case "enabled":
			if enabled, ok := setting.Value.(bool); ok {
				currentConfig.AudioProcessing.MakeupGain.Enabled = enabled
			}
		case "gain":
			if gain, ok := setting.Value.(string); ok {
				if val, err := strconv.ParseFloat(gain, 32); err == nil {
					currentConfig.AudioProcessing.MakeupGain.GainDB = float32(val)
				}
			}
		}
	}
	
	// Set preset to custom when individual settings change
	currentConfig.AudioProcessing.Preset = "custom"
	
	// Update UI state
	webTUI.Lock()
	webTUI.AudioPreset = "custom"
	webTUI.Unlock()
	
	// Apply to processor immediately
	applyAudioConfigToProcessor(currentConfig)
	
	// Save config to file
	if err := saveClientConfig("settings.config", currentConfig); err != nil {
		LogError("Failed to save audio setting: %v", err)
	} else {
		LogDebug("Audio setting saved: %s.%s = %v", setting.Section, setting.Param, setting.Value)
	}
	
	broadcastUpdate()
}

// Test microphone handler
func handleTestMicrophone() {
	LogInfo("Testing microphone audio levels")
	appState.AddMessage("🎤 Testing microphone - speak now!", "info")
	
	// Trigger audio pipeline test
	go func() {
		TestAudioPipeline()
		time.Sleep(2 * time.Second)
		appState.AddMessage("Microphone test completed", "success")
	}()
}

// Save custom preset handler  
func handleSaveCustomPreset() {
	if currentConfig == nil {
		LogError("No config to save custom preset")
		appState.AddMessage("Error: No configuration loaded", "error")
		return
	}
	
	LogInfo("Saving custom audio preset")
	
	// Save current settings as custom preset
	if err := saveClientConfig("settings.config", currentConfig); err != nil {
		LogError("Failed to save custom preset: %v", err)
		appState.AddMessage("Failed to save custom preset", "error")
	} else {
		LogInfo("Custom audio preset saved successfully")
		appState.AddMessage("💾 Custom preset saved!", "success")
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		LogError("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	wsMutex.Lock()
	wsClients[conn] = true
	wsMutex.Unlock()
	
	// Send initial state
	webTUI.RLock()
	initialState := *webTUI
	webTUI.RUnlock()
	conn.WriteJSON(initialState)
	
	// Keep connection alive and handle disconnection
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wsMutex.Lock()
			delete(wsClients, conn)
			wsMutex.Unlock()
			break
		}
	}
}

func broadcastUpdate() {
	webTUI.RLock()
	state := *webTUI
	webTUI.RUnlock()
	
	wsMutex.Lock()
	defer wsMutex.Unlock()
	
	for client := range wsClients {
		if err := client.WriteJSON(state); err != nil {
			client.Close()
			delete(wsClients, client)
		}
	}
}

// LEGACY WebTUI functions - these still exist for backward compatibility during transition
// But now they're mostly redundant since WebTUI is an observer

func WebTUISetConnected(connected bool, nickname, serverName, motd string) {
	// Still doing dual updates during transition
	if connected {
		appState.AddMessage(fmt.Sprintf("Connected as %s", nickname), "success")
		WebTUIAddMessage(fmt.Sprintf("Connected as %s", nickname), "success")
		
		appState.AddMessage(motd, "info")
		WebTUIAddMessage(motd, "info")
	} else {
		appState.AddMessage("Disconnected from server", "error")
		WebTUIAddMessage("Disconnected from server", "error")
	}
}

func WebTUISetChannel(channel string) {
	// Still doing dual updates during transition
	appState.AddMessage(fmt.Sprintf("Joined channel: %s", channel), "success")
	WebTUIAddMessage(fmt.Sprintf("Joined channel: %s", channel), "success")
}

func WebTUISetChannels(channels []string) {
	// Observer handles this now, but keeping function for compatibility
}

func WebTUISetChannelUsers(channelUsers map[string][]string) {
	// Observer handles this now, but keeping function for compatibility
}

func WebTUISetPTT(active bool) {
	// Observer handles this now, but keeping function for compatibility
}

func WebTUISetAudioLevel(level int) {
	// Observer handles this now, but keeping function for compatibility
}

func WebTUIIncrementRX() {
	// Observer handles this now, but keeping function for compatibility
}

func WebTUIIncrementTX() {
	// Observer handles this now, but keeping function for compatibility
}

func WebTUIAddMessage(message, msgType string) {
	// Legacy function - still used during transition
	webTUI.Lock()
	timestamp := time.Now().Format("15:04:05")
	webMsg := WebMessage{
		Timestamp: timestamp,
		Message:   message,
		Type:      msgType,
	}
	webTUI.Messages = append(webTUI.Messages, webMsg)
	
	// Keep only last 100 messages
	if len(webTUI.Messages) > 100 {
		webTUI.Messages = webTUI.Messages[len(webTUI.Messages)-100:]
	}
	webTUI.Unlock()
	
	broadcastUpdate()
}

func WebTUISetPTTKey(keyName string) {
	// Observer handles this now, but keeping function for compatibility
}