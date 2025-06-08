// FILE: client/main.go
package main

import (
	"ahcli/common/logger"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/gordonklaus/portaudio"
)

func main() {
	// Initialize unified logging system FIRST
	err := logger.Init("client")
	if err != nil {
		fmt.Printf("Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Enable debug mode for development
	logger.SetDebugMode(true)

	logger.Info("=== AHCLI Client Starting ===")
	logger.Info("Log file: %s", logger.GetLogPath())

	// Initialize application state
	InitAppState()
	logger.Debug("Application state initialized")

	// Initialize PortAudio globally
	err = portaudio.Initialize()
	if err != nil {
		logger.Fatal("PortAudio init failed: %v", err)
		return
	}
	defer portaudio.Terminate()
	logger.Info("PortAudio initialized successfully")

	// Load config
	config, err := loadClientConfig("settings.config")
	if err != nil {
		logger.Fatal("Error loading config: %v", err)
		return
	}

	// Store config reference for audio controls
	currentConfig = config
	logger.Info("Client config loaded successfully")

	// Log audio processing settings
	logger.Info("Audio preset: %s", config.AudioProcessing.Preset)
	logger.Debug("Noise gate: enabled=%t, threshold=%.1fdB",
		config.AudioProcessing.NoiseGate.Enabled,
		config.AudioProcessing.NoiseGate.ThresholdDB)
	logger.Debug("Compressor: enabled=%t, threshold=%.1fdB, ratio=%.1f",
		config.AudioProcessing.Compressor.Enabled,
		config.AudioProcessing.Compressor.ThresholdDB,
		config.AudioProcessing.Compressor.Ratio)
	logger.Debug("Makeup gain: enabled=%t, gain=%.1fdB",
		config.AudioProcessing.MakeupGain.Enabled,
		config.AudioProcessing.MakeupGain.GainDB)

	// Set PTT key from config
	pttKeyCode = keyNameToVKCode(config.PTTKey)
	if pttKeyCode == 0 {
		logger.Fatal("Unsupported PTT key: %s", config.PTTKey)
		return
	}

	StartPTTListener()
	logger.Info("PTT listener started (key: %s)", config.PTTKey)

	// Initialize client crypto system
	err = InitClientCrypto()
	if err != nil {
		logger.Fatal("Failed to initialize crypto system: %v", err)
		return
	}
	if clientCrypto == nil {
		logger.Fatal("CLIENT CRYPTO IS NIL!")
		return
	} else {
		logger.Info("Client crypto system initialized successfully")
	}

	// Initialize audio system
	logger.Info("Initializing audio system...")
	err = InitAudio()
	if err != nil {
		logger.Fatal("Audio initialization failed: %v", err)
		return
	}
	logger.Info("Audio initialized successfully")

	// Apply audio config to processor AFTER audio init
	applyAudioConfigToProcessor(config)
	logger.Info("Audio processing settings applied from config")

	// Initialize Web UI server
	port, err := StartWebServer()
	if err != nil {
		logger.Fatal("Web server failed: %v", err)
		return
	}
	logger.Info("Web server started on port %d", port)

	// PURE APPSTATE: Only update AppState - observer handles WebTUI
	appState.SetPTTKey(config.PTTKey)

	// Welcome messages - PURE APPSTATE only
	appState.AddMessage("AHCLI Voice Chat ready!", "info")
	appState.AddMessage(fmt.Sprintf("Hold %s to transmit", config.PTTKey), "info")
	appState.AddMessage("Right-click system tray to open UI", "info")

	// Create hidden window for tray messages
	err = createHiddenWindow()
	if err != nil {
		logger.Fatal("Failed to create hidden window: %v", err)
		return
	}
	logger.Debug("Hidden window created for tray message handling")

	// Initialize system tray
	err = InitTray(port)
	if err != nil {
		logger.Fatal("Failed to initialize system tray: %v", err)
		return
	}
	logger.Info("System tray initialized")

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
	logger.Debug("AppState observer registered for tray icon updates")

	// Test audio pipeline
	go func() {
		time.Sleep(3 * time.Second)
		TestAudioPipeline()
	}()

	// Start connection in background
	go func() {
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		serverAddr := config.Servers[config.PreferredServer].IP
		appState.AddMessage(fmt.Sprintf("Connecting to %s...", serverAddr), "info")
		logger.Info("Attempting connection to %s", serverAddr)

		err := connectToServer(config)
		if err != nil {
			logger.Error("Connection error: %v", err)

			// PURE APPSTATE: Only update AppState - observer handles WebTUI
			appState.AddMessage(fmt.Sprintf("Connection error: %s", err.Error()), "error")

			time.Sleep(2 * time.Second)
			os.Exit(1)
		}
	}()

	logger.Info("AHCLI running in background - check system tray")
	logger.Info("Left-click tray icon to open UI, right-click for menu")
	logger.Info("🎯 UNIFIED LOGGING MIGRATION COMPLETE - All systems now use common/logger!")

	// Auto-launch UI on startup
	go func() {
		time.Sleep(1 * time.Second) // Wait for tray to settle
		openVoiceChatUI()           // Launch browser automatically
	}()

	// Run Windows message loop
	runMessageLoop()
}

// createHiddenWindow creates an invisible window to receive tray messages
func createHiddenWindow() error {
	logger.Debug("Creating hidden window for tray message handling")

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
		logger.Error("Failed to register window class")
		return fmt.Errorf("failed to register window class")
	}

	// Create hidden window
	hwnd, _, _ = createWindowEx.Call(
		0,                                  // dwExStyle
		uintptr(unsafe.Pointer(className)), // lpClassName
		0,                                  // lpWindowName
		0,                                  // dwStyle
		0, 0, 0, 0,                         // x, y, width, height
		0,         // hWndParent
		0,         // hMenu
		hInstance, // hInstance
		0,         // lpParam
	)

	if hwnd == 0 {
		logger.Error("Failed to create hidden window")
		return fmt.Errorf("failed to create hidden window")
	}

	logger.Debug("Hidden window created successfully")
	return nil
}

// windowProc handles Windows messages for our hidden window
func windowProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		logger.Debug("Received tray icon message: %d", lParam)
		HandleTrayMessage(lParam)
		return 0
	default:
		ret, _, _ := defWindowProc.Call(hwnd, msg, wParam, lParam)
		return ret
	}
}

// runMessageLoop runs the Windows message loop
func runMessageLoop() {
	logger.Debug("Starting Windows message loop")

	var msg MSG
	for {
		bRet, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if bRet == 0 { // WM_QUIT
			logger.Debug("Received WM_QUIT message")
			break
		} else if bRet == 1 { // Regular message
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		// bRet == -1 is error, but we'll continue
	}

	// Cleanup before exit
	CleanupTray()
	logger.Info("Message loop ended, AHCLI shutting down")
}
