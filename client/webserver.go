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
	
	return port, nil
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
		WebTUIAddMessage(fmt.Sprintf("Joining channel: %s", cmd.Args), "info")
	case "quit":
		WebTUIAddMessage("Disconnecting...", "info")
		// Could trigger graceful shutdown here
	default:
		WebTUIAddMessage(fmt.Sprintf("Unknown command: %s", cmd.Command), "error")
	}
	
	w.WriteHeader(200)
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

// Web TUI update functions
func WebTUISetConnected(connected bool, nickname, serverName, motd string) {
	webTUI.Lock()
	webTUI.Connected = connected
	webTUI.Nickname = nickname
	webTUI.ServerName = serverName
	if connected {
		webTUI.ConnectionTime = time.Now()
	}
	webTUI.Unlock()
	
	if connected {
		WebTUIAddMessage(fmt.Sprintf("Connected as %s", nickname), "success")
		WebTUIAddMessage(motd, "info")
	} else {
		WebTUIAddMessage("Disconnected from server", "error")
	}
	
	broadcastUpdate()
}

func WebTUISetChannel(channel string) {
	webTUI.Lock()
	webTUI.CurrentChannel = channel
	webTUI.Unlock()
	
	WebTUIAddMessage(fmt.Sprintf("Joined channel: %s", channel), "success")
	broadcastUpdate()
}

func WebTUISetChannels(channels []string) {
	webTUI.Lock()
	webTUI.Channels = channels
	webTUI.Unlock()
	
	broadcastUpdate()
}

func WebTUISetChannelUsers(channelUsers map[string][]string) {
	webTUI.Lock()
	webTUI.ChannelUsers = channelUsers
	webTUI.Unlock()
	
	broadcastUpdate()
}

func WebTUISetPTT(active bool) {
	webTUI.Lock()
	webTUI.PTTActive = active
	webTUI.Unlock()
	
	// Only broadcast PTT changes occasionally to avoid spam
	// We'll handle this with debouncing in the web UI
	broadcastUpdate()
}

func WebTUISetAudioLevel(level int) {
	webTUI.Lock()
	webTUI.AudioLevel = level
	webTUI.Unlock()
	
	// Don't broadcast every audio level change - too spammy
	// Web UI will poll for this or we'll batch updates
}

func WebTUIIncrementRX() {
	webTUI.Lock()
	webTUI.PacketsRx++
	webTUI.Unlock()
	
	// Batch network updates - only broadcast every 10 packets
	if webTUI.PacketsRx%10 == 0 {
		broadcastUpdate()
	}
}

func WebTUIIncrementTX() {
	webTUI.Lock()
	webTUI.PacketsTx++
	webTUI.Unlock()
	
	// Batch network updates
	if webTUI.PacketsTx%10 == 0 {
		broadcastUpdate()
	}
}

func WebTUIAddMessage(message, msgType string) {
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
	webTUI.Lock()
	webTUI.PTTKey = keyName
	webTUI.Unlock()
	
	broadcastUpdate()
}