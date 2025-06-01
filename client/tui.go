package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TUIState struct {
	sync.RWMutex
	connected      bool
	nickname       string
	currentChannel string
	users          []string
	channels       []string
	channelUsers   map[string][]string // channel -> users in that channel
	serverName     string
	motd           string
	pttActive      bool
	audioLevel     int
	lastActivity   time.Time
	connectionTime time.Time
	packetsRx      int
	packetsTx      int
	errorCount     int
	lastError      string
}

var (
	tuiState = &TUIState{
		lastActivity:   time.Now(),
		connectionTime: time.Now(),
		channelUsers:   make(map[string][]string),
	}
	app               *tview.Application
	statusView        *tview.TextView
	usersView         *tview.TextView
	chatView          *tview.TextView
	inputView         *tview.InputField
	headerView        *tview.TextView
	audioView         *tview.TextView
	currentPTTKeyName string = "LSHIFT" // Default, will be updated by config
)

func InitTUI() error {
	app = tview.NewApplication()

	// Set dark theme colors similar to the terminal screenshot
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorNavy
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorDarkBlue
	tview.Styles.BorderColor = tcell.ColorDarkCyan
	tview.Styles.TitleColor = tcell.ColorLightCyan
	tview.Styles.GraphicsColor = tcell.ColorDarkCyan
	tview.Styles.PrimaryTextColor = tcell.ColorLightCyan
	tview.Styles.SecondaryTextColor = tcell.ColorDarkCyan
	tview.Styles.TertiaryTextColor = tcell.ColorGray
	tview.Styles.InverseTextColor = tcell.ColorBlack
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorDarkCyan

	// Create header
	headerView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	headerView.SetBorder(true).SetTitle(" AHCLI Voice Chat ").SetTitleAlign(tview.AlignCenter)

	// Create status panel (left side)
	statusView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	statusView.SetBorder(true).SetTitle(" Status ").SetTitleAlign(tview.AlignLeft)

	// Create channel/users list (right side) - this is the key change
	usersView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	usersView.SetBorder(true).SetTitle(" Channels & Users ").SetTitleAlign(tview.AlignLeft)

	// Create audio level indicator (right side, below users)
	audioView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	audioView.SetBorder(true).SetTitle(" Audio ").SetTitleAlign(tview.AlignLeft)

	// Create chat/log area (center)
	chatView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	chatView.SetBorder(true).SetTitle(" Messages ").SetTitleAlign(tview.AlignLeft)

	// Create input field (bottom)
	inputView = tview.NewInputField().
		SetLabel("> ").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				handleTUIInput(inputView.GetText())
				inputView.SetText("")
			}
		})
	inputView.SetBorder(true).SetTitle(" Commands ").SetTitleAlign(tview.AlignLeft)

	// Create layout
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(usersView, 0, 2, false).
		AddItem(audioView, 8, 0, false)

	mainArea := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(chatView, 0, 1, false).
		AddItem(inputView, 3, 0, true)

	contentArea := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(statusView, 25, 0, false).
		AddItem(mainArea, 0, 2, true).
		AddItem(rightPanel, 30, 0, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(headerView, 3, 0, false).
		AddItem(contentArea, 0, 1, false)

	app.SetRoot(root, true)

	// Start update goroutine
	go updateTUI()

	return nil
}

func StartTUI() error {
	LogInfo("Starting TUI...")
	return app.Run()
}

func StopTUI() {
	app.Stop()
}

func updateTUI() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		app.QueueUpdateDraw(func() {
			updateHeader()
			updateStatus()
			updateUsers()
			updateAudio()
		})
	}
}

func updateHeader() {
	tuiState.RLock()
	defer tuiState.RUnlock()

	status := "[red]Disconnected[-]"
	if tuiState.connected {
		status = "[lightcyan]Connected[-]"
	}

	serverInfo := ""
	if tuiState.serverName != "" {
		serverInfo = fmt.Sprintf(" • Server: [yellow]%s[-]", tuiState.serverName)
	}

	channelInfo := ""
	if tuiState.currentChannel != "" {
		channelInfo = fmt.Sprintf(" • Channel: [lightblue]%s[-]", tuiState.currentChannel)
	}

	headerText := fmt.Sprintf("%s%s%s", status, serverInfo, channelInfo)
	headerView.SetText(headerText)
}

func updateStatus() {
	tuiState.RLock()
	defer tuiState.RUnlock()

	var sb strings.Builder

	// Connection info
	sb.WriteString("[white::b]Connection:[-]\n")
	if tuiState.connected {
		sb.WriteString(fmt.Sprintf("[lightcyan]Connected[-] as [yellow]%s[-]\n", tuiState.nickname))
		uptime := time.Since(tuiState.connectionTime).Round(time.Second)
		sb.WriteString(fmt.Sprintf("Uptime: [lightblue]%s[-]\n", uptime))
	} else {
		sb.WriteString("[red]Disconnected[-]\n")
	}

	sb.WriteString("\n")

	// Network stats
	sb.WriteString("[white::b]Network:[-]\n")
	sb.WriteString(fmt.Sprintf("RX: [lightcyan]%d[-] packets\n", tuiState.packetsRx))
	sb.WriteString(fmt.Sprintf("TX: [yellow]%d[-] packets\n", tuiState.packetsTx))
	if tuiState.errorCount > 0 {
		sb.WriteString(fmt.Sprintf("Errors: [red]%d[-]\n", tuiState.errorCount))
	}

	sb.WriteString("\n")

	// PTT status
	sb.WriteString("[white::b]PTT:[-]\n")
	if tuiState.pttActive {
		sb.WriteString("[red::b]TRANSMITTING[-]\n")
	} else {
		sb.WriteString("[darkcyan]Ready[-]\n")
	}

	sb.WriteString("\n")

	// MOTD
	if tuiState.motd != "" {
		sb.WriteString("[white::b]MOTD:[-]\n")
		sb.WriteString(fmt.Sprintf("[darkcyan]%s[-]\n", tuiState.motd))
	}

	// Last error
	if tuiState.lastError != "" {
		sb.WriteString("\n[white::b]Last Error:[-]\n")
		sb.WriteString(fmt.Sprintf("[red]%s[-]", tuiState.lastError))
	}

	statusView.SetText(sb.String())
}

func updateUsers() {
	tuiState.RLock()
	defer tuiState.RUnlock()

	var sb strings.Builder

	// Show channels with users nested underneath (Ventrilo style)
	if len(tuiState.channels) == 0 {
		sb.WriteString("[darkcyan]No channels available[-]")
	} else {
		for _, channel := range tuiState.channels {
			// Channel header
			if channel == tuiState.currentChannel {
				// Current channel highlighted
				sb.WriteString(fmt.Sprintf("[lightblue::b]▶ %s[-]\n", channel))
			} else {
				// Other channels
				sb.WriteString(fmt.Sprintf("[white::b]▷ %s[-]\n", channel))
			}

			// Users in this channel
			usersInChannel, exists := tuiState.channelUsers[channel]
			if exists && len(usersInChannel) > 0 {
				for _, user := range usersInChannel {
					if user == tuiState.nickname {
						// Current user highlighted
						sb.WriteString(fmt.Sprintf("  [yellow]├─ %s (you)[-]\n", user))
					} else {
						// Other users
						sb.WriteString(fmt.Sprintf("  [lightcyan]├─ %s[-]\n", user))
					}
				}
			} else {
				// Empty channel
				sb.WriteString("  [darkcyan]├─ (empty)[-]\n")
			}
			sb.WriteString("\n")
		}
	}

	usersView.SetText(sb.String())
}

func updateAudio() {
	tuiState.RLock()
	level := tuiState.audioLevel
	ptt := tuiState.pttActive
	tuiState.RUnlock()

	var sb strings.Builder

	// PTT indicator
	if ptt {
		sb.WriteString("[red::b]● TRANSMITTING[-]\n\n")
	} else {
		sb.WriteString("[darkcyan]○ Ready[-]\n\n")
	}

	// Audio level bar
	sb.WriteString("Level:\n")
	barLength := 15
	filledBars := int(float64(level) / 100.0 * float64(barLength))

	sb.WriteString("[")
	for i := 0; i < barLength; i++ {
		if i < filledBars {
			if i < barLength/3 {
				sb.WriteString("[lightcyan]█[-]")
			} else if i < 2*barLength/3 {
				sb.WriteString("[yellow]█[-]")
			} else {
				sb.WriteString("[red]█[-]")
			}
		} else {
			sb.WriteString("[darkcyan]░[-]")
		}
	}
	sb.WriteString("]\n")

	sb.WriteString(fmt.Sprintf("%d%%", level))

	audioView.SetText(sb.String())
}

func AddChatMessage(message string) {
	timestamp := time.Now().Format("15:04:05")
	fullMessage := fmt.Sprintf("[darkcyan]%s[-] %s", timestamp, message)

	app.QueueUpdateDraw(func() {
		fmt.Fprintf(chatView, "%s\n", fullMessage)
	})
}

func handleTUIInput(input string) {
	if input == "" {
		return
	}

	AddChatMessage(fmt.Sprintf("[yellow]> %s[-]", input))

	// Handle commands
	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		if len(parts) == 0 {
			return
		}

		command := strings.ToLower(parts[0])
		switch command {
		case "/join":
			if len(parts) >= 2 {
				channel := parts[1]
				// Send channel change request (you'll need to implement this)
				changeChannel(channel)
				AddChatMessage(fmt.Sprintf("[lightblue]Requesting to join channel: %s[-]", channel))
			} else {
				AddChatMessage("[red]Usage: /join <channel>[-]")
			}
		case "/quit", "/exit":
			app.Stop()
		case "/help":
			showHelp()
		default:
			AddChatMessage(fmt.Sprintf("[red]Unknown command: %s[-]", command))
			AddChatMessage("[darkcyan]Type /help for available commands[-]")
		}
	}
}

func showHelp() {
	AddChatMessage("[white::b]Available Commands:[-]")
	AddChatMessage("[lightblue]/join <channel>[-] - Join a channel")
	AddChatMessage("[lightblue]/quit[-] - Exit the application")
	AddChatMessage("[lightblue]/help[-] - Show this help")
	// Get PTT key from config dynamically
	AddChatMessage(fmt.Sprintf("[darkcyan]Hold %s to transmit audio[-]", getCurrentPTTKey()))
}

// Helper function to get current PTT key name
func getCurrentPTTKey() string {
	// This will be set by main.go when config is loaded
	return currentPTTKeyName
}

// TUI State update functions (call these from your existing code)
func TUISetConnected(connected bool, nickname, serverName, motd string) {
	tuiState.Lock()
	tuiState.connected = connected
	tuiState.nickname = nickname
	tuiState.serverName = serverName
	tuiState.motd = motd
	if connected {
		tuiState.connectionTime = time.Now()
	}
	tuiState.Unlock()

	if connected {
		AddChatMessage(fmt.Sprintf("[lightcyan]Connected as %s[-]", nickname))
		AddChatMessage(fmt.Sprintf("[darkcyan]%s[-]", motd))
	} else {
		AddChatMessage("[red]Disconnected from server[-]")
	}
}

func TUISetChannel(channel string) {
	tuiState.Lock()
	tuiState.currentChannel = channel
	tuiState.Unlock()

	AddChatMessage(fmt.Sprintf("[lightblue]Joined channel: %s[-]", channel))
}

func TUISetUsers(users []string) {
	tuiState.Lock()
	tuiState.users = users
	tuiState.Unlock()
}

func TUISetChannels(channels []string) {
	tuiState.Lock()
	tuiState.channels = channels
	tuiState.Unlock()
}

// New function to set users per channel (Ventrilo style)
func TUISetChannelUsers(channelUsers map[string][]string) {
	tuiState.Lock()
	tuiState.channelUsers = channelUsers
	tuiState.Unlock()
}

// Helper to add a user to a specific channel
func TUIAddUserToChannel(channel, user string) {
	tuiState.Lock()
	if tuiState.channelUsers[channel] == nil {
		tuiState.channelUsers[channel] = make([]string, 0)
	}
	
	// Check if user already exists in channel
	for _, existingUser := range tuiState.channelUsers[channel] {
		if existingUser == user {
			tuiState.Unlock()
			return
		}
	}
	
	tuiState.channelUsers[channel] = append(tuiState.channelUsers[channel], user)
	tuiState.Unlock()
}

// Helper to remove a user from a specific channel
func TUIRemoveUserFromChannel(channel, user string) {
	tuiState.Lock()
	users := tuiState.channelUsers[channel]
	for i, u := range users {
		if u == user {
			tuiState.channelUsers[channel] = append(users[:i], users[i+1:]...)
			break
		}
	}
	tuiState.Unlock()
}

func TUISetPTT(active bool) {
	tuiState.Lock()
	tuiState.pttActive = active
	tuiState.Unlock()
}

func TUISetAudioLevel(level int) {
	tuiState.Lock()
	tuiState.audioLevel = level
	tuiState.Unlock()
}

func TUIIncrementRX() {
	tuiState.Lock()
	tuiState.packetsRx++
	tuiState.Unlock()
}

func TUIIncrementTX() {
	tuiState.Lock()
	tuiState.packetsTx++
	tuiState.Unlock()
}

func TUISetError(err string) {
	tuiState.Lock()
	tuiState.errorCount++
	tuiState.lastError = err
	tuiState.Unlock()

	AddChatMessage(fmt.Sprintf("[red]Error: %s[-]", err))
}

// Set PTT key name for display in TUI
func TUISetPTTKey(keyName string) {
	currentPTTKeyName = keyName
}