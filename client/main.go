// FILE: client/main.go

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gordonklaus/portaudio"
)

var (
	noTUI = flag.Bool("no-tui", false, "Disable TUI and use console output")
)

// Helper function to check if TUI is disabled
func isTUIDisabled() bool {
	return *noTUI
}

func main() {
	// Parse flags before initializing logger
	flag.Parse()

	// Initialize logging (parses --debug flag)
	InitLogger()
	defer CloseLogger()

	// Initialize PortAudio globally
	err := portaudio.Initialize()
	if err != nil {
		if isTUIDisabled() {
			println("PortAudio init failed:", err.Error())
		}
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
		if isTUIDisabled() {
			println("Error loading config:", err.Error())
			println("Looking for config at:", configPath)
		}
		LogError("Error loading config: %v", err)
		LogError("Config path attempted: %s", configPath)
		return
	}

	LogInfo("Client config loaded successfully from: %s", configPath)

	// Set PTT key from config
	pttKeyCode = keyNameToVKCode(config.PTTKey)
	if pttKeyCode == 0 {
		if isTUIDisabled() {
			println("Unsupported PTT key:", config.PTTKey)
		}
		LogError("Unsupported PTT key: %s", config.PTTKey)
		return
	}

	StartPTTListener()

	// Initialize audio system
	LogMain("Initializing audio...")
	err = InitAudio()
	if err != nil {
		if isTUIDisabled() {
			println("Audio initialization failed:", err.Error())
		}
		LogError("Audio initialization failed: %v", err)
		return
	}
	LogMain("Audio initialized successfully")

	// Initialize Web UI (unless disabled)
	if !isTUIDisabled() {
		port, err := StartWebServer()
		if err != nil {
			LogError("Web server failed: %v", err)
			fmt.Println("Web server failed:", err)
			return
		}
		
		WebTUISetPTTKey(config.PTTKey)
		WebTUIAddMessage("Welcome to AHCLI Voice Chat!", "info")
		WebTUIAddMessage(fmt.Sprintf("Hold %s to transmit audio", config.PTTKey), "info")
		WebTUIAddMessage(fmt.Sprintf("Connecting to %s...", config.Servers[config.PreferredServer].IP), "info")
		
		// Launch Chromium in app mode
		go launchChromiumApp(port)
	}

	// Test audio pipeline (optional)
	go func() {
		time.Sleep(3 * time.Second)
		TestAudioPipeline()
	}()

	// Start connection in background
	go func() {
		err := connectToServer(config)
		if err != nil {
			if isTUIDisabled() {
				println("Connection error:", err.Error())
			} else {
				ConsoleTUIAddMessage(fmt.Sprintf("Error: %s", err.Error()))
			}
			LogError("Connection error: %v", err)
			
			// Exit if connection fails
			if !isTUIDisabled() {
				time.Sleep(2 * time.Second)
			}
			os.Exit(1)
		}
	}()

	// Handle user input
	if !isTUIDisabled() {
		// Web UI handles input through HTTP API
		LogInfo("Web UI started, waiting for connections...")
		select {} // Keep running
	} else {
		// Console mode - just wait
		select {}
	}
}

func launchChromiumApp(port int) {
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
	
	// Fallback to system Chrome/Edge
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