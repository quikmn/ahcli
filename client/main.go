// FILE: client/main.go

package main

import (
	"fmt"
	"os"
	"time"
	"syscall"
	"unsafe"

	"github.com/gordonklaus/portaudio"
)

func main() {
	// Initialize logging
	InitLogger()
	defer CloseLogger()

	// Initialize application state
	InitAppState()
	LogInfo("Application state initialized")

	// Initialize PortAudio globally
	err := portaudio.Initialize()
	if err != nil {
		LogError("PortAudio init failed: %v", err)
		return
	}
	defer portaudio.Terminate()

	// Load config
	config, err := loadClientConfig("settings.config")
	if err != nil {
		LogError("Error loading config: %v", err)
		return
	}

	LogInfo("Client config loaded successfully")

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

	// Initialize Web UI server
	port, err := StartWebServer()
	if err != nil {
		LogError("Web server failed: %v", err)
		fmt.Println("Web server failed:", err)
		return
	}
	LogInfo("Web server started on port %d", port)

	// Set initial state in both systems
	appState.SetPTTKey(config.PTTKey)
	WebTUISetPTTKey(config.PTTKey)
	
	// Welcome messages
	appState.AddMessage("AHCLI Voice Chat ready!", "info")
	WebTUIAddMessage("AHCLI Voice Chat ready!", "info")
	
	appState.AddMessage(fmt.Sprintf("Hold %s to transmit", config.PTTKey), "info")
	WebTUIAddMessage(fmt.Sprintf("Hold %s to transmit", config.PTTKey), "info")
	
	appState.AddMessage("Right-click system tray to open UI", "info")
	WebTUIAddMessage("Right-click system tray to open UI", "info")

	// Create hidden window for tray messages
	err = createHiddenWindow()
	if err != nil {
		LogError("Failed to create hidden window: %v", err)
		return
	}

	// Initialize system tray
	err = InitTray(port)
	if err != nil {
		LogError("Failed to initialize system tray: %v", err)
		return
	}
	LogInfo("System tray initialized")

	// Set up AppState observer to update tray when connection changes
	appState.AddObserver(func(change StateChange) {
		if change.Type == "connection" {
			if data, ok := change.Data.(map[string]interface{}); ok {
				if connected, ok := data["connected"].(bool); ok {
					UpdateTrayIcon(connected)
				}
			}
		}
	})

	// Test audio pipeline
	go func() {
		time.Sleep(3 * time.Second)
		TestAudioPipeline()
	}()

	// Start connection in background
	go func() {
		appState.AddMessage(fmt.Sprintf("Connecting to %s...", config.Servers[config.PreferredServer].IP), "info")
		WebTUIAddMessage(fmt.Sprintf("Connecting to %s...", config.Servers[config.PreferredServer].IP), "info")
		
		err := connectToServer(config)
		if err != nil {
			LogError("Connection error: %v", err)
			
			appState.AddMessage(fmt.Sprintf("Connection error: %s", err.Error()), "error")
			WebTUIAddMessage(fmt.Sprintf("Connection error: %s", err.Error()), "error")
			
			time.Sleep(2 * time.Second)
			os.Exit(1)
		}
	}()

	LogInfo("AHCLI running in background - check system tray")
	LogInfo("Left-click tray icon to open UI, right-click for menu")

	// Auto-launch UI on startup for immediate access
	go func() {
		time.Sleep(1 * time.Second) // Wait for tray to settle
		openVoiceChatUI()           // Launch browser automatically
	}()

	// Run Windows message loop
	runMessageLoop()
}

// createHiddenWindow creates an invisible window to receive tray messages
func createHiddenWindow() error {
	hInstance, _, _ := getModuleHandle.Call(0)
	
	className := syscall.StringToUTF16Ptr("AHCLITrayWindow")
	
	// Register window class
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(windowProc),
		HInstance:     hInstance,
		LpszClassName: className,
	}
	
	atom, _, _ := registerClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("failed to register window class")
	}
	
	// Create hidden window
	hwnd, _, _ = createWindowEx.Call(
		0,                    // dwExStyle
		uintptr(unsafe.Pointer(className)), // lpClassName
		0,                    // lpWindowName
		0,                    // dwStyle
		0, 0, 0, 0,          // x, y, width, height
		0,                    // hWndParent
		0,                    // hMenu
		hInstance,            // hInstance
		0,                    // lpParam
	)
	
	if hwnd == 0 {
		return fmt.Errorf("failed to create hidden window")
	}
	
	return nil
}

// windowProc handles Windows messages for our hidden window
func windowProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		HandleTrayMessage(lParam)
		return 0
	default:
		ret, _, _ := defWindowProc.Call(hwnd, msg, wParam, lParam)
		return ret
	}
}

// runMessageLoop runs the Windows message loop
func runMessageLoop() {
	var msg MSG
	for {
		bRet, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if bRet == 0 { // WM_QUIT
			break
		} else if bRet == 1 { // Regular message
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		// bRet == -1 is error, but we'll continue
	}
	
	// Cleanup before exit
	CleanupTray()
	LogInfo("Message loop ended, AHCLI shutting down")
}