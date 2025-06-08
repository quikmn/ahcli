// FILE: client/webserver.go
package main

import (
	"ahcli/common/logger"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed web/*
var webFiles embed.FS

type WebTUIState struct {
	sync.RWMutex
	Connected      bool                `json:"connected"`
	Nickname       string              `json:"nickname"`
	ServerName     string              `json:"serverName"`
	CurrentChannel string              `json:"currentChannel"`
	Channels       []string            `json:"channels"`
	ChannelUsers   map[string][]string `json:"channelUsers"`
	PTTActive      bool                `json:"pttActive"`
	AudioLevel     int                 `json:"audioLevel"`
	PacketsRx      int                 `json:"packetsRx"`
	PacketsTx      int                 `json:"packetsTx"`
	ConnectionTime time.Time           `json:"connectionTime"`
	Messages       []WebMessage        `json:"messages"`
	PTTKey         string              `json:"pttKey"`

	// Real-time audio processing stats
	AudioPreset   string  `json:"audioPreset"`
	InputLevel    float32 `json:"inputLevel"`
	OutputLevel   float32 `json:"outputLevel"`
	GateOpen      bool    `json:"gateOpen"`
	GainReduction float32 `json:"gainReduction"`
	AudioQuality  string  `json:"audioQuality"`

	// Detailed processing stats for debugging
	NoiseGateThreshold float32 `json:"noiseGateThreshold"`
	CompressorRatio    float32 `json:"compressorRatio"`
	MakeupGainDB       float32 `json:"makeupGainDB"`

	RawInputLevel       float32 `json:"rawInputLevel"`
	ProcessedInputLevel float32 `json:"processedInputLevel"`
	BypassProcessing    bool    `json:"bypassProcessing"`
}

type WebMessage struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Type      string `json:"type"` // "info", "error", "success", "ptt", "chat"
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
	wsClients      = make(map[*websocket.Conn]bool)
	wsMutex        sync.Mutex
	observersSetup = false

	// Global config reference for audio controls
	currentConfig *ClientConfig
)

func StartWebServer() (int, error) {
	// Find available port
	port := findAvailablePort(8080)
	logger.Debug("Found available port: %d", port)

	// Serve embedded files with proper routing
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		logger.Error("Failed to create web filesystem: %v", err)
		return 0, err
	}
	http.Handle("/", http.FileServer(http.FS(webFS)))
	logger.Debug("Web filesystem configured with embedded files")

	// API endpoints
	http.HandleFunc("/api/state", handleAPIState)
	http.HandleFunc("/api/command", handleAPICommand)
	http.HandleFunc("/ws", handleWebSocket)
	logger.Debug("Web API endpoints registered")

	logger.Info("Starting web server on port %d", port)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			logger.Error("Web server failed: %v", err)
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

	logger.Info("Setting up WebTUI as AppState observer...")

	appState.AddObserver(func(change StateChange) {
		switch change.Type {
		case "ptt":
			if active, ok := change.Data.(bool); ok {
				logger.Debug("Observer: PTT state changed to %t", active)
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
				logger.Debug("Observer: Connection state changed")
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
				logger.Debug("Observer: Channel changed to %s", channel)
				webTUI.Lock()
				webTUI.CurrentChannel = channel
				webTUI.Unlock()
				broadcastUpdate()
			}

		case "channels":
			if channels, ok := change.Data.([]string); ok {
				logger.Debug("Observer: Channels list updated")
				webTUI.Lock()
				webTUI.Channels = channels
				webTUI.Unlock()
				broadcastUpdate()
			}

		case "channel_users":
			if channelUsers, ok := change.Data.(map[string][]string); ok {
				logger.Debug("Observer: Channel users updated")
				webTUI.Lock()
				webTUI.ChannelUsers = channelUsers
				webTUI.Unlock()
				broadcastUpdate()
			}

		case "message":
			if msg, ok := change.Data.(AppMessage); ok {
				logger.Debug("Observer: New message - %s", msg.Message)
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
				logger.Debug("Observer: PTT key changed to %s", keyName)
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

		// Audio processing stats observer
		case "audio_stats":
			if stats, ok := change.Data.(AudioStats); ok {
				logger.Debug("Observer: Audio stats updated - Input: %.1f%%, Gate: %t, Quality: %s",
					stats.InputLevel*100, stats.NoiseGateOpen, stats.AudioQuality)

				webTUI.Lock()
				webTUI.InputLevel = stats.InputLevel
				webTUI.OutputLevel = stats.InputLevel // For now, use input level for both
				webTUI.GateOpen = stats.NoiseGateOpen
				webTUI.GainReduction = 1.0 - stats.CompressionGain // Convert to reduction amount
				webTUI.AudioQuality = stats.AudioQuality

				// Update current processing settings for UI display
				if audioProcessor != nil {
					webTUI.NoiseGateThreshold = audioProcessor.noiseGate.threshold
					webTUI.CompressorRatio = audioProcessor.compressor.ratio
					webTUI.MakeupGainDB = audioProcessor.makeupGain.gainDB
				}
				webTUI.Unlock()

				// Broadcast audio stats updates (but not too frequently)
				broadcastUpdate()
			}

		// Real-time input level updates
		case "input_level":
			if level, ok := change.Data.(float32); ok {
				webTUI.Lock()
				webTUI.InputLevel = level
				// Also update the legacy AudioLevel field for backward compatibility
				webTUI.AudioLevel = int(level * 100)
				webTUI.Unlock()
				// Don't broadcast every input level - too frequent
			}

		// Noise gate status updates
		case "gate_status":
			if open, ok := change.Data.(bool); ok {
				webTUI.Lock()
				webTUI.GateOpen = open
				webTUI.Unlock()
				broadcastUpdate()
			}
		}
	})

	observersSetup = true
	logger.Info("WebTUI observers setup complete - now pure observer of AppState!")
}

func findAvailablePort(startPort int) int {
	logger.Debug("Searching for available port starting from %d", startPort)

	for port := startPort; port < startPort+100; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			logger.Debug("Found available port: %d", port)
			return port
		}
	}

	logger.Warn("No available ports found, falling back to %d", startPort)
	return startPort // fallback
}

func handleAPIState(w http.ResponseWriter, r *http.Request) {
	logger.Debug("API state request from %s", r.RemoteAddr)

	webTUI.RLock()
	defer webTUI.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webTUI)
}

func handleAPICommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		logger.Debug("API command rejected: method %s not allowed", r.Method)
		http.Error(w, "Method not allowed", 405)
		return
	}

	var cmd struct {
		Command string `json:"command"`
		Args    string `json:"args"`
	}

	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		logger.Error("Invalid JSON in API command: %v", err)
		http.Error(w, "Invalid JSON", 400)
		return
	}

	logger.Info("API command received: %s with args: %s", cmd.Command, cmd.Args)

	switch cmd.Command {
	case "join":
		changeChannel(cmd.Args)
		appState.AddMessage(fmt.Sprintf("Joining channel: %s", cmd.Args), "info")

	case "quit":
		logger.Info("Quit command received from web interface")
		appState.AddMessage("Disconnecting...", "info")
		// Could trigger graceful shutdown here

	case "audio_preset":
		handleAudioPreset(cmd.Args)

	case "audio_setting":
		handleAudioSetting(cmd.Args)

	case "bypass_processing":
		handleBypassToggle(cmd.Args)

	case "test_microphone":
		handleTestMicrophone()

	case "save_custom_preset":
		handleSaveCustomPreset()

	case "chat":
		// NEW: Handle chat messages from UI
		handleChatCommand(cmd.Args)

	default:
		logger.Error("Unknown API command: %s", cmd.Command)
		appState.AddMessage(fmt.Sprintf("Unknown command: %s", cmd.Command), "error")
	}

	w.WriteHeader(200)
}

// NEW: Handle chat messages from the web UI
func handleChatCommand(message string) {
	if message == "" {
		logger.Debug("Empty chat message received, ignoring")
		return
	}

	logger.Info("Web UI chat message: %s", message)

	// Send to server via the network layer
	sendChatMessage(message)

	// Note: We don't add the message locally here
	// The server will broadcast it back to us, which creates the proper flow
}

// Audio preset handler
func handleAudioPreset(preset string) {
	logger.Info("Changing audio preset to: %s", preset)

	if currentConfig == nil {
		logger.Error("No config loaded for audio preset change")
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
		logger.Error("Failed to save audio preset: %v", err)
		appState.AddMessage("Failed to save audio settings", "error")
	} else {
		logger.Info("Audio preset '%s' applied and saved", preset)
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
		logger.Error("Invalid audio setting JSON: %v", err)
		return
	}

	logger.Info("Updating audio setting: %s.%s = %v", setting.Section, setting.Param, setting.Value)

	if currentConfig == nil {
		logger.Error("No config loaded for audio setting change")
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
		logger.Error("Failed to save audio setting: %v", err)
	} else {
		logger.Debug("Audio setting saved: %s.%s = %v", setting.Section, setting.Param, setting.Value)
	}

	broadcastUpdate()
}

// Test microphone handler
func handleTestMicrophone() {
	logger.Info("Testing microphone audio levels")
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
		logger.Error("No config to save custom preset")
		appState.AddMessage("Error: No configuration loaded", "error")
		return
	}

	logger.Info("Saving custom audio preset")

	// Save current settings as custom preset
	if err := saveClientConfig("settings.config", currentConfig); err != nil {
		logger.Error("Failed to save custom preset: %v", err)
		appState.AddMessage("Failed to save custom preset", "error")
	} else {
		logger.Info("Custom audio preset saved successfully")
		appState.AddMessage("💾 Custom preset saved!", "success")
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger.Debug("WebSocket connection attempt from %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	wsMutex.Lock()
	wsClients[conn] = true
	clientCount := len(wsClients)
	wsMutex.Unlock()

	logger.Info("WebSocket client connected from %s (total: %d)", r.RemoteAddr, clientCount)

	// Send initial state
	webTUI.RLock()
	initialState := *webTUI
	webTUI.RUnlock()

	if err := conn.WriteJSON(initialState); err != nil {
		logger.Error("Failed to send initial state to WebSocket client: %v", err)
		return
	}

	// Keep connection alive and handle disconnection
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wsMutex.Lock()
			delete(wsClients, conn)
			remainingClients := len(wsClients)
			wsMutex.Unlock()

			logger.Debug("WebSocket client disconnected from %s (remaining: %d)", r.RemoteAddr, remainingClients)
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

	activeClients := 0
	for client := range wsClients {
		if err := client.WriteJSON(state); err != nil {
			logger.Debug("WebSocket client write failed, removing: %v", err)
			client.Close()
			delete(wsClients, client)
		} else {
			activeClients++
		}
	}

	if activeClients > 0 {
		logger.Debug("Broadcasted update to %d WebSocket clients", activeClients)
	}
}

// LEGACY WebTUI functions - keeping for backward compatibility during transition

func WebTUISetConnected(connected bool, nickname, serverName, motd string) {
	// Still doing dual updates during transition
	if connected {
		appState.AddMessage(fmt.Sprintf("Connected as %s", nickname), "success")
	} else {
		appState.AddMessage("Disconnected from server", "error")
	}
}

func WebTUISetChannel(channel string) {
	// Still doing dual updates during transition
	appState.AddMessage(fmt.Sprintf("Joined channel: %s", channel), "success")
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

// Handle bypass processing toggle
func handleBypassToggle(args string) {
	bypass := args == "true"

	logger.Info("Setting audio processing bypass to: %t", bypass)

	if audioProcessor == nil {
		logger.Error("Audio processor not initialized")
		appState.AddMessage("Error: Audio processor not ready", "error")
		return
	}

	// Set bypass in processor
	audioProcessor.SetBypass(bypass)

	// Update AppState
	appState.SetBypassProcessing(bypass)

	// User feedback
	if bypass {
		appState.AddMessage("🔀 Audio processing BYPASSED", "warning")
	} else {
		appState.AddMessage("⚙️ Audio processing ACTIVE", "success")
	}

	logger.Info("Audio processing bypass set to: %t", bypass)
}
