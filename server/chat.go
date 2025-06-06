// FILE: server/chat.go

package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ChatMessage represents a single chat message
type ChatMessage struct {
	GUID      string    `json:"guid"`      // Channel GUID for routing
	Channel   string    `json:"channel"`   // Human-readable channel name
	Username  string    `json:"username"`  // User who sent the message
	Message   string    `json:"message"`   // The actual message content
	Timestamp time.Time `json:"timestamp"` // When the message was sent
}

// ChatStorage manages all chat functionality
type ChatStorage struct {
	sync.RWMutex

	// In-memory storage: GUID -> []ChatMessage
	messages map[string][]ChatMessage

	// Configuration
	enabled      bool
	logFile      string
	maxMessages  int
	recentOnJoin int

	// Log file handle
	logFileHandle *os.File
}

// Global chat storage instance
var chatStorage *ChatStorage

// InitChatStorage initializes the chat system
func InitChatStorage(config *ServerConfig) error {
	if !config.Chat.Enabled {
		LogInfo("Chat system disabled in configuration")
		return nil
	}

	chatStorage = &ChatStorage{
		messages:     make(map[string][]ChatMessage),
		enabled:      config.Chat.Enabled,
		logFile:      config.Chat.LogFile,
		maxMessages:  config.Chat.MaxMessages,
		recentOnJoin: config.Chat.LoadRecentOnJoin,
	}

	// Open log file for append-only writing
	var err error
	chatStorage.logFileHandle, err = os.OpenFile(chatStorage.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open chat log file: %v", err)
	}

	// Generate GUIDs for channels that don't have them
	err = chatStorage.ensureChannelGUIDs(config)
	if err != nil {
		return fmt.Errorf("failed to generate channel GUIDs: %v", err)
	}

	// Load existing chat history from log file
	err = chatStorage.loadHistoryFromLog()
	if err != nil {
		LogError("Failed to load chat history: %v", err)
		// Don't fail initialization, just log the error
	}

	LogInfo("Chat system initialized - log file: %s, max messages: %d", chatStorage.logFile, chatStorage.maxMessages)
	return nil
}

// ensureChannelGUIDs generates GUIDs for channels that don't have them
func (cs *ChatStorage) ensureChannelGUIDs(config *ServerConfig) error {
	needsUpdate := false

	for i := range config.Channels {
		if config.Channels[i].GUID == "" {
			guid, err := generateGUID()
			if err != nil {
				return err
			}
			config.Channels[i].GUID = guid
			needsUpdate = true
			LogInfo("Generated GUID for channel '%s': %s", config.Channels[i].Name, guid)
		}
	}

	// Save config if we generated new GUIDs
	if needsUpdate {
		err := saveServerConfig("config.json", config)
		if err != nil {
			LogError("Failed to save config with new GUIDs: %v", err)
			// Don't fail, GUIDs are still in memory
		} else {
			LogInfo("Saved config with generated GUIDs")
		}
	}

	return nil
}

// generateGUID creates a simple UUID-like identifier
func generateGUID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Format as UUID-like string
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

// StoreMessage stores a chat message and writes it to the log
func (cs *ChatStorage) StoreMessage(guid, channel, username, message string) error {
	if !cs.enabled {
		return nil
	}

	chatMsg := ChatMessage{
		GUID:      guid,
		Channel:   channel,
		Username:  username,
		Message:   message,
		Timestamp: time.Now(),
	}

	cs.Lock()
	defer cs.Unlock()

	// Add to in-memory storage
	if cs.messages[guid] == nil {
		cs.messages[guid] = make([]ChatMessage, 0)
	}
	cs.messages[guid] = append(cs.messages[guid], chatMsg)

	// Implement circular buffer - drop oldest messages if we exceed max
	if len(cs.messages[guid]) > cs.maxMessages {
		// Keep the newest messages
		keepFrom := len(cs.messages[guid]) - (cs.maxMessages - 10000) // Drop 10k when limit reached
		cs.messages[guid] = cs.messages[guid][keepFrom:]
		LogDebug("Circular buffer: dropped old messages for channel %s, now have %d messages", channel, len(cs.messages[guid]))
	}

	// Write to log file
	err := cs.writeToLog(chatMsg)
	if err != nil {
		LogError("Failed to write chat message to log: %v", err)
		// Don't fail the store operation, message is still in memory
	}

	LogDebug("Stored chat message in %s (%s): <%s> %s", channel, guid, username, message)
	return nil
}

// writeToLog writes a message to the append-only log file
func (cs *ChatStorage) writeToLog(msg ChatMessage) error {
	if cs.logFileHandle == nil {
		return fmt.Errorf("log file not open")
	}

	// Log format: 2025-06-03T05:25:30Z [guid:a1b2c3d4] [General] <username> message
	logLine := fmt.Sprintf("%s [guid:%s] [%s] <%s> %s\n",
		msg.Timestamp.UTC().Format(time.RFC3339),
		msg.GUID,
		msg.Channel,
		msg.Username,
		msg.Message)

	_, err := cs.logFileHandle.WriteString(logLine)
	if err != nil {
		return err
	}

	// Flush to ensure it's written immediately
	return cs.logFileHandle.Sync()
}

// GetRecentMessages returns recent messages for a channel GUID
func (cs *ChatStorage) GetRecentMessages(guid string, count int) []ChatMessage {
	if !cs.enabled {
		return nil
	}

	cs.RLock()
	defer cs.RUnlock()

	messages, exists := cs.messages[guid]
	if !exists || len(messages) == 0 {
		return nil
	}

	// Return the last 'count' messages
	if len(messages) <= count {
		// Return all messages if we have fewer than requested
		result := make([]ChatMessage, len(messages))
		copy(result, messages)
		return result
	}

	// Return the last 'count' messages
	startIdx := len(messages) - count
	result := make([]ChatMessage, count)
	copy(result, messages[startIdx:])
	return result
}

// loadHistoryFromLog loads chat history from the log file on startup
func (cs *ChatStorage) loadHistoryFromLog() error {
	if cs.logFile == "" {
		return nil
	}

	file, err := os.Open(cs.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			LogInfo("Chat log file doesn't exist yet, starting fresh")
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	loadedCount := 0

	for scanner.Scan() {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		msg, err := cs.parseLogLine(line)
		if err != nil {
			LogDebug("Failed to parse log line %d: %v", lineCount, err)
			continue
		}

		// Add to in-memory storage (without writing back to log)
		if cs.messages[msg.GUID] == nil {
			cs.messages[msg.GUID] = make([]ChatMessage, 0)
		}
		cs.messages[msg.GUID] = append(cs.messages[msg.GUID], *msg)
		loadedCount++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Sort messages by timestamp for each channel
	for guid := range cs.messages {
		sort.Slice(cs.messages[guid], func(i, j int) bool {
			return cs.messages[guid][i].Timestamp.Before(cs.messages[guid][j].Timestamp)
		})
	}

	LogInfo("Loaded %d chat messages from log file (%d lines processed)", loadedCount, lineCount)
	return nil
}

// parseLogLine parses a log line back into a ChatMessage
func (cs *ChatStorage) parseLogLine(line string) (*ChatMessage, error) {
	// Expected format: 2025-06-03T05:25:30Z [guid:a1b2c3d4] [General] <username> message

	// Parse timestamp
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid log line format")
	}

	timestamp, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %v", err)
	}

	remaining := parts[1]

	// Parse GUID
	if !strings.HasPrefix(remaining, "[guid:") {
		return nil, fmt.Errorf("missing GUID section")
	}

	guidEnd := strings.Index(remaining, "]")
	if guidEnd == -1 {
		return nil, fmt.Errorf("malformed GUID section")
	}

	guid := remaining[6:guidEnd] // Skip "[guid:"
	remaining = strings.TrimSpace(remaining[guidEnd+1:])

	// Parse channel name
	if !strings.HasPrefix(remaining, "[") {
		return nil, fmt.Errorf("missing channel section")
	}

	channelEnd := strings.Index(remaining, "]")
	if channelEnd == -1 {
		return nil, fmt.Errorf("malformed channel section")
	}

	channel := remaining[1:channelEnd] // Skip "["
	remaining = strings.TrimSpace(remaining[channelEnd+1:])

	// Parse username
	if !strings.HasPrefix(remaining, "<") {
		return nil, fmt.Errorf("missing username section")
	}

	usernameEnd := strings.Index(remaining, ">")
	if usernameEnd == -1 {
		return nil, fmt.Errorf("malformed username section")
	}

	username := remaining[1:usernameEnd] // Skip "<"
	message := strings.TrimSpace(remaining[usernameEnd+1:])

	return &ChatMessage{
		GUID:      guid,
		Channel:   channel,
		Username:  username,
		Message:   message,
		Timestamp: timestamp,
	}, nil
}

// GetChannelGUID returns the GUID for a channel name
func GetChannelGUID(channelName string) string {
	for _, channel := range serverConfig.Channels {
		if channel.Name == channelName {
			return channel.GUID
		}
	}
	return ""
}

// GetChannelName returns the name for a channel GUID
func GetChannelName(guid string) string {
	for _, channel := range serverConfig.Channels {
		if channel.GUID == guid {
			return channel.Name
		}
	}
	return ""
}

// CloseChatStorage properly closes the chat storage system
func CloseChatStorage() {
	if chatStorage != nil && chatStorage.logFileHandle != nil {
		chatStorage.logFileHandle.Close()
		LogInfo("Chat storage closed")
	}
}

// saveServerConfig saves the server configuration to a file
func saveServerConfig(path string, config *ServerConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
