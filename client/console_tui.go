// client/console_tui.go
//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type ConsoleTUIState struct {
	sync.RWMutex
	connected      bool
	nickname       string
	serverName     string
	currentChannel string
	channels       []string
	channelUsers   map[string][]string
	pttActive      bool
	audioLevel     int
	packetsRx      int
	packetsTx      int
	connectionTime time.Time
	lastUpdate     time.Time
	messages       []string
}

var (
	consoleTUI = &ConsoleTUIState{
		channelUsers:   make(map[string][]string),
		messages:       make([]string, 0),
		connectionTime: time.Now(),
	}
	pttKeyName = "LSHIFT" // Will be set from config
)

func InitConsoleTUI() {
	// Set console title
	exec.Command("cmd", "/c", "title", "AHCLI Voice Chat").Run()
	
	// Initial draw
	consoleTUI.Lock()
	consoleTUI.lastUpdate = time.Now()
	consoleTUI.Unlock()
	
	drawInterface()
	
	// Start update loop - only redraw when state changes
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		lastState := getStateHash()
		for range ticker.C {
			currentState := getStateHash()
			if currentState != lastState {
				drawInterface()
				lastState = currentState
			}
		}
	}()
}

func getStateHash() string {
	consoleTUI.RLock()
	defer consoleTUI.RUnlock()
	
	// Simple hash of current state
	return fmt.Sprintf("%v|%s|%s|%v|%d|%d", 
		consoleTUI.connected, 
		consoleTUI.currentChannel, 
		strings.Join(consoleTUI.channels, ","),
		consoleTUI.pttActive,
		consoleTUI.packetsRx,
		consoleTUI.packetsTx)
}

func drawInterface() {
	consoleTUI.RLock()
	defer consoleTUI.RUnlock()
	
	// Clear screen
	exec.Command("cmd", "/c", "cls").Run()
	
	// Header - made wider
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║                                                AHCLI Voice Chat                                                       ║\n")
	
	status := "Disconnected"
	if consoleTUI.connected {
		status = fmt.Sprintf("Connected to %s", consoleTUI.serverName)
	}
	fmt.Printf("║ Status: %-100s ║\n", status)
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╣")
	
	// Three column layout: Connection | Messages | Channels & Users (adjusted proportions)
	fmt.Println("║ CONNECTION      │ MESSAGES                                           │ CHANNELS & USERS              ║")
	fmt.Println("╟─────────────────┼────────────────────────────────────────────────────┼───────────────────────────────╢")
	
	// Draw 15 content rows with new column widths
	for i := 0; i < 15; i++ {
		statusCol := getStatusLine(i)
		messageCol := getMessageLine(i)
		channelCol := getChannelLine(i)
		
		// Adjusted column widths: 15 | 50 | 29
		statusTrunc := truncate(statusCol, 15)
		messageTrunc := truncate(messageCol, 50)
		channelTrunc := truncate(channelCol, 29)
		
		fmt.Printf("║ %-15s │ %-50s │ %-29s ║\n", statusTrunc, messageTrunc, channelTrunc)
	}
	
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╣")
	
	// PTT Status - wider
	pttStatus := "○ Ready"
	if consoleTUI.pttActive {
		pttStatus = "● TRANSMITTING"
	}
	fmt.Printf("║ PTT: %-12s │ Audio Level: %-15s │ RX: %-8d TX: %-8d │ Network Stats ║\n", 
		pttStatus, 
		getAudioBar(),
		consoleTUI.packetsRx,
		consoleTUI.packetsTx)
	
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Commands: /join <channel>, /quit, /help | Hold %s to transmit\n", pttKeyName)
	fmt.Print("> ")
}

func getStatusLine(line int) string {
	switch line {
	case 0:
		if consoleTUI.connected {
			return fmt.Sprintf("User: %s", consoleTUI.nickname)
		}
		return "Not connected"
	case 1:
		if consoleTUI.connected {
			uptime := time.Since(consoleTUI.connectionTime).Round(time.Second)
			return fmt.Sprintf("Uptime: %s", uptime)
		}
		return ""
	case 2:
		return ""
	case 3:
		return "NETWORK:"
	case 4:
		return fmt.Sprintf("RX: %d pkts", consoleTUI.packetsRx)
	case 5:
		return fmt.Sprintf("TX: %d pkts", consoleTUI.packetsTx)
	case 6:
		return ""
	case 7:
		return "CHANNEL:"
	case 8:
		if consoleTUI.currentChannel != "" {
			return consoleTUI.currentChannel
		}
		return "None"
	default:
		return ""
	}
}

func getMessageLine(line int) string {
	// Don't lock here - caller should already have the lock
	if line < len(consoleTUI.messages) {
		// Show latest messages first (reverse order)
		idx := len(consoleTUI.messages) - 1 - line
		if idx >= 0 && idx < len(consoleTUI.messages) {
			return consoleTUI.messages[idx]
		}
	}
	return ""
}

func getChannelLine(line int) string {
	lineCount := 0
	
	// Draw channels with users nested underneath
	for _, channel := range consoleTUI.channels {
		if lineCount == line {
			prefix := "▷ "
			if channel == consoleTUI.currentChannel {
				prefix = "▶ "
			}
			return prefix + channel
		}
		lineCount++
		
		// Users in this channel
		users, exists := consoleTUI.channelUsers[channel]
		if exists {
			for _, user := range users {
				if lineCount == line {
					suffix := ""
					if user == consoleTUI.nickname {
						suffix = " (you)"
					}
					return "  ├─ " + user + suffix
				}
				lineCount++
			}
		} else {
			if lineCount == line {
				return "  ├─ (empty)"
			}
			lineCount++
		}
		
		// Add spacing between channels
		if lineCount == line {
			return ""
		}
		lineCount++
	}
	
	return ""
}

func getAudioBar() string {
	level := consoleTUI.audioLevel
	barLength := 10
	filledBars := int(float64(level) / 100.0 * float64(barLength))
	
	bar := ""
	for i := 0; i < barLength; i++ {
		if i < filledBars {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("%s %d%%", bar, level)
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// Console TUI update functions
func ConsoleTUISetConnected(connected bool, nickname, serverName, motd string) {
	consoleTUI.Lock()
	consoleTUI.connected = connected
	consoleTUI.nickname = nickname
	consoleTUI.serverName = serverName
	if connected {
		consoleTUI.connectionTime = time.Now()
	}
	consoleTUI.Unlock()
	
	if connected {
		ConsoleTUIAddMessage(fmt.Sprintf("Connected as %s", nickname))
		ConsoleTUIAddMessage(motd)
	} else {
		ConsoleTUIAddMessage("Disconnected")
	}
}

func ConsoleTUISetChannel(channel string) {
	consoleTUI.Lock()
	consoleTUI.currentChannel = channel
	consoleTUI.Unlock()
	ConsoleTUIAddMessage(fmt.Sprintf("Joined: %s", channel))
}

func ConsoleTUISetChannels(channels []string) {
	consoleTUI.Lock()
	consoleTUI.channels = channels
	consoleTUI.Unlock()
}

func ConsoleTUISetChannelUsers(channelUsers map[string][]string) {
	consoleTUI.Lock()
	consoleTUI.channelUsers = channelUsers
	consoleTUI.Unlock()
}

func ConsoleTUISetPTT(active bool) {
	consoleTUI.Lock()
	consoleTUI.pttActive = active
	consoleTUI.Unlock()
}

func ConsoleTUISetAudioLevel(level int) {
	consoleTUI.Lock()
	consoleTUI.audioLevel = level
	consoleTUI.Unlock()
}

func ConsoleTUIIncrementRX() {
	consoleTUI.Lock()
	consoleTUI.packetsRx++
	consoleTUI.Unlock()
}

func ConsoleTUIIncrementTX() {
	consoleTUI.Lock()
	consoleTUI.packetsTx++
	consoleTUI.Unlock()
}

func ConsoleTUIAddMessage(message string) {
	consoleTUI.Lock()
	timestamp := time.Now().Format("15:04:05")
	fullMessage := fmt.Sprintf("[%s] %s", timestamp, message)
	consoleTUI.messages = append(consoleTUI.messages, fullMessage)
	
	// Keep only last 50 messages
	if len(consoleTUI.messages) > 50 {
		consoleTUI.messages = consoleTUI.messages[len(consoleTUI.messages)-50:]
	}
	consoleTUI.Unlock()
}

func ConsoleTUISetPTTKey(keyName string) {
	pttKeyName = keyName
}