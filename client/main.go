// FILE: client/main.go

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gordonklaus/portaudio"
)

func main() {
	// Initialize logging
	InitLogger()
	defer CloseLogger()

	// Initialize application state - THE NEW HOTNESS
	InitAppState()
	LogInfo("Application state initialized")

	// Initialize PortAudio globally
	err := portaudio.Initialize()
	if err != nil {
		LogError("PortAudio init failed: %v", err)
		return
	}
	defer portaudio.Terminate()

	// Get executable directory for config file
	exePath, err := os.Executable()
	if err != nil {
		LogError("Failed to get executable path: %v", err)
		exePath = ""
	}
	configDir := filepath.Dir(exePath)
	configPath := filepath.Join(configDir, "settings.config")

	// Try current directory if exe dir doesn't have config (for development)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "settings.config"
	}

	config, err := loadClientConfig(configPath)
	if err != nil {
		LogError("Error loading config: %v", err)
		LogError("Config path attempted: %s", configPath)
		return
	}

	LogInfo("Client config loaded successfully from: %s", configPath)

	// Set PTT key from config
	pttKeyCode = keyNameToVKCode(config.PTTKey)
	if pttKeyCode == 0 {
		LogError("Unsupported PTT key: %s", config.PTTKey)
		return
	}

	StartPTTListener()

	// Initialize audio system
	LogInfo("Initializing audio system...")
	err = InitAudio()
	if err != nil {
		LogError("Audio initialization failed: %v", err)
		return
	}
	LogInfo("Audio initialized successfully")

	// Initialize Web UI
	port, err := StartWebServer()
	if err != nil {
		LogError("Web server failed: %v", err)
		fmt.Println("Web server failed:", err)
		return
	}
	
	// Set initial state in AppState as well as WebTUI
	appState.SetPTTKey(config.PTTKey)
	WebTUISetPTTKey(config.PTTKey)
	
	// Welcome messages to both systems
	appState.AddMessage("Welcome to AHCLI Voice Chat!", "info")
	WebTUIAddMessage("Welcome to AHCLI Voice Chat!", "info")
	
	appState.AddMessage(fmt.Sprintf("Hold %s to transmit audio", config.PTTKey), "info")
	WebTUIAddMessage(fmt.Sprintf("Hold %s to transmit audio", config.PTTKey), "info")
	
	appState.AddMessage(fmt.Sprintf("Connecting to %s...", config.Servers[config.PreferredServer].IP), "info")
	WebTUIAddMessage(fmt.Sprintf("Connecting to %s...", config.Servers[config.PreferredServer].IP), "info")
	
	// Launch browser
	go launchBrowser(port)

	// Test audio pipeline (optional)
	go func() {
		time.Sleep(3 * time.Second)
		TestAudioPipeline()
	}()

	// Start connection in background
	go func() {
		err := connectToServer(config)
		if err != nil {
			LogError("Connection error: %v", err)
			
			// Update both systems
			appState.AddMessage(fmt.Sprintf("Connection error: %s", err.Error()), "error")
			WebTUIAddMessage(fmt.Sprintf("Connection error: %s", err.Error()), "error")
			
			// Exit if connection fails
			time.Sleep(2 * time.Second)
			os.Exit(1)
		}
	}()

	// Web UI handles all interaction through HTTP API
	LogInfo("Web UI started, waiting for connections...")
	LogInfo("AppState bridge is active - dual-write pattern ready!")
	select {} // Keep running
}

func launchBrowser(port int) {
	// Wait for web server to be ready
	time.Sleep(2 * time.Second)
	
	url := fmt.Sprintf("http://localhost:%d", port)
	
	// Try bundled Chromium first
	chromiumPath := "./chromium/launch-app.bat"
	if _, err := os.Stat(chromiumPath); err == nil {
		LogInfo("Launching bundled Chromium...")
		cmd := exec.Command("cmd", "/c", chromiumPath, strconv.Itoa(port))
		cmd.Start()
		return
	}
	
	// Fallback to system browsers
	browsers := [][]string{
		{"chrome", "--app=" + url, "--disable-web-security", "--disable-features=TranslateUI"},
		{"msedge", "--app=" + url, "--disable-web-security"},
		{"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe", "--app=" + url},
		{"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe", "--app=" + url},
	}
	
	for _, browser := range browsers {
		cmd := exec.Command(browser[0], browser[1:]...)
		if err := cmd.Start(); err == nil {
			LogInfo("Launched browser: %s", browser[0])
			return
		}
	}
	
	// Final fallback - default browser
	LogInfo("Opening in default browser...")
	exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}