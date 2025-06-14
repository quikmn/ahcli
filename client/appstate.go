package main

import (
	"sync"
	"time"
)

// StateChange represents a change in application state
type StateChange struct {
	Type string      // "ptt", "audio_level", "connection", "channel", "message"
	Data interface{} // The actual state data
}

// Observer function type for state changes
type StateObserver func(StateChange)

// AppState manages all application state in a centralized, thread-safe way
type AppState struct {
	mutex sync.RWMutex

	// Audio state
	PTTActive  bool
	AudioLevel int
	PacketsRx  int
	PacketsTx  int

	// Connection state
	Connected      bool
	Nickname       string
	ServerName     string
	MOTD           string
	ConnectionTime time.Time

	// Channel state
	CurrentChannel string
	Channels       []string
	ChannelUsers   map[string][]string

	// UI state
	PTTKey   string
	Messages []AppMessage

	// Observer pattern for UI updates
	observers []StateObserver

	RawInputLevel       float32 // Before any processing
	ProcessedInputLevel float32 // After processing
	BypassProcessing    bool    // Bypass toggle state
}

// AppMessage represents a message in the application
type AppMessage struct {
	Timestamp string
	Message   string
	Type      string // "info", "error", "success", "ptt"
}

// Global state instance
var appState *AppState

// InitAppState initializes the global application state
func InitAppState() {
	appState = &AppState{
		ChannelUsers: make(map[string][]string),
		Messages:     make([]AppMessage, 0),
		PTTKey:       "LSHIFT",
		observers:    make([]StateObserver, 0),
	}
}

// AddObserver adds a function that will be called when state changes
func (as *AppState) AddObserver(observer StateObserver) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	as.observers = append(as.observers, observer)
}

// notifyObservers sends state change notifications to all observers
func (as *AppState) notifyObservers(changeType string, data interface{}) {
	as.mutex.RLock()
	observers := make([]StateObserver, len(as.observers))
	copy(observers, as.observers)
	as.mutex.RUnlock()

	change := StateChange{
		Type: changeType,
		Data: data,
	}

	// Call observers without holding the lock
	for _, observer := range observers {
		observer(change)
	}
}

// === AUDIO STATE METHODS ===

// SetRawInputLevel updates raw input level
func (as *AppState) SetRawInputLevel(level float32) {
	as.mutex.Lock()
	as.RawInputLevel = level
	as.mutex.Unlock()
	as.notifyObservers("raw_input_level", level)
}

// SetProcessedInputLevel updates processed input level
func (as *AppState) SetProcessedInputLevel(level float32) {
	as.mutex.Lock()
	as.ProcessedInputLevel = level
	as.mutex.Unlock()
	as.notifyObservers("processed_input_level", level)
}

// SetBypassProcessing updates bypass state
func (as *AppState) SetBypassProcessing(bypass bool) {
	as.mutex.Lock()
	as.BypassProcessing = bypass
	as.mutex.Unlock()
	as.notifyObservers("bypass_processing", bypass)
}

// GetProcessedInputLevel returns current processed level
func (as *AppState) GetProcessedInputLevel() float32 {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.ProcessedInputLevel
}

// SetPTTActive updates PTT state and notifies observers
func (as *AppState) SetPTTActive(active bool) {
	as.mutex.Lock()
	if as.PTTActive != active {
		as.PTTActive = active
		as.mutex.Unlock()
		as.notifyObservers("ptt", active)
	} else {
		as.mutex.Unlock()
	}
}

// GetPTTActive returns current PTT state
func (as *AppState) GetPTTActive() bool {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return as.PTTActive
}

// SetAudioLevel updates audio level and notifies observers
func (as *AppState) SetAudioLevel(level int) {
	as.mutex.Lock()
	as.AudioLevel = level
	as.mutex.Unlock()
	as.notifyObservers("audio_level", level)
}

// IncrementRX increments received packet counter
func (as *AppState) IncrementRX() {
	as.mutex.Lock()
	as.PacketsRx++
	packets := as.PacketsRx
	as.mutex.Unlock()

	// Only notify every 10 packets to avoid spam
	if packets%10 == 0 {
		as.notifyObservers("packets_rx", packets)
	}
}

// IncrementTX increments transmitted packet counter
func (as *AppState) IncrementTX() {
	as.mutex.Lock()
	as.PacketsTx++
	packets := as.PacketsTx
	as.mutex.Unlock()

	// Only notify every 10 packets to avoid spam
	if packets%10 == 0 {
		as.notifyObservers("packets_tx", packets)
	}
}

// === CONNECTION STATE METHODS ===

// SetConnected updates connection state
func (as *AppState) SetConnected(connected bool, nickname, serverName, motd string) {
	as.mutex.Lock()
	as.Connected = connected
	as.Nickname = nickname
	as.ServerName = serverName
	as.MOTD = motd
	if connected {
		as.ConnectionTime = time.Now()
	}
	as.mutex.Unlock()

	connectionData := map[string]interface{}{
		"connected":  connected,
		"nickname":   nickname,
		"serverName": serverName,
		"motd":       motd,
	}
	as.notifyObservers("connection", connectionData)
}

// === CHANNEL STATE METHODS ===

// SetChannel updates current channel
func (as *AppState) SetChannel(channel string) {
	as.mutex.Lock()
	as.CurrentChannel = channel
	as.mutex.Unlock()
	as.notifyObservers("channel", channel)
}

// SetChannels updates available channels list
func (as *AppState) SetChannels(channels []string) {
	as.mutex.Lock()
	as.Channels = channels
	as.mutex.Unlock()
	as.notifyObservers("channels", channels)
}

// SetChannelUsers updates channel user lists
func (as *AppState) SetChannelUsers(channelUsers map[string][]string) {
	as.mutex.Lock()
	as.ChannelUsers = channelUsers
	as.mutex.Unlock()
	as.notifyObservers("channel_users", channelUsers)
}

// === MESSAGE METHODS ===

// AddMessage adds a message and notifies observers
func (as *AppState) AddMessage(message, msgType string) {
	timestamp := time.Now().Format("15:04:05")
	msg := AppMessage{
		Timestamp: timestamp,
		Message:   message,
		Type:      msgType,
	}

	as.mutex.Lock()
	as.Messages = append(as.Messages, msg)

	// Keep only last 100 messages
	if len(as.Messages) > 100 {
		as.Messages = as.Messages[len(as.Messages)-100:]
	}
	as.mutex.Unlock()

	as.notifyObservers("message", msg)
}

// SetPTTKey updates PTT key setting
func (as *AppState) SetPTTKey(keyName string) {
	as.mutex.Lock()
	as.PTTKey = keyName
	as.mutex.Unlock()
	as.notifyObservers("ptt_key", keyName)
}

// === NEW AUDIO VISUALIZATION METHODS ===

// SetAudioStats updates comprehensive audio processing statistics
func (as *AppState) SetAudioStats(stats AudioStats) {
	// Don't store stats in AppState to keep it clean
	// Just forward to observers for UI updates
	go as.notifyObservers("audio_stats", stats)
}

// SetInputLevel updates real-time input level (0.0 to 1.0)
func (as *AppState) SetInputLevel(level float32) {
	as.mutex.Lock()
	// Convert to 0-100 range for existing AudioLevel field (backward compatibility)
	as.AudioLevel = int(level * 100)
	as.mutex.Unlock()

	// Send high-frequency updates for smooth visualization
	go as.notifyObservers("input_level", level)
}

// SetGateStatus updates noise gate open/closed status
func (as *AppState) SetGateStatus(open bool) {
	// Send instant updates for immediate visual feedback
	go as.notifyObservers("gate_status", open)
}

// GetInputLevel returns current input level (thread-safe)
func (as *AppState) GetInputLevel() float32 {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	return float32(as.AudioLevel) / 100.0
}

// === CONVENIENCE METHODS ===

// GetState returns a snapshot of current state (thread-safe)
func (as *AppState) GetState() map[string]interface{} {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	return map[string]interface{}{
		"connected":      as.Connected,
		"nickname":       as.Nickname,
		"serverName":     as.ServerName,
		"currentChannel": as.CurrentChannel,
		"channels":       as.Channels,
		"channelUsers":   as.ChannelUsers,
		"pttActive":      as.PTTActive,
		"audioLevel":     as.AudioLevel,
		"packetsRx":      as.PacketsRx,
		"packetsTx":      as.PacketsTx,
		"connectionTime": as.ConnectionTime,
		"messages":       as.Messages,
		"pttKey":         as.PTTKey,
	}
}
