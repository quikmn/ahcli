//go:build windows

// FILE: client/tray.go
package main

import (
	"ahcli/common/logger"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	webServerPort int
)

// InitTray initializes the system tray icon
func InitTray(port int) error {
	webServerPort = port
	logger.Info("Initializing system tray icon on port %d", port)

	// Load custom application icon
	logger.Debug("Attempting to load custom icon: ahcli.ico")
	hIcon, _, _ := loadImage.Call(
		0, // hInstance (0 for loading from file)
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ahcli.ico"))),
		1, // IMAGE_ICON
		0, // cxDesired (0 = default size)
		0, // cyDesired (0 = default size)
		LR_LOADFROMFILE,
	)

	// Fallback to default icon if custom icon fails to load
	if hIcon == 0 {
		logger.Debug("Custom icon not found, using default system icon")
		hInstance, _, _ := getModuleHandle.Call(0)
		hIcon, _, _ = loadIcon.Call(hInstance, uintptr(32512)) // IDI_APPLICATION
	} else {
		logger.Debug("Custom icon loaded successfully")
	}

	// Create tray icon
	nid := NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		Hwnd:             hwnd,
		UID:              trayIconID,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            hIcon,
	}

	// Set tooltip
	copy(nid.SzTip[:], syscall.StringToUTF16("AHCLI Voice Chat"))

	ret, _, _ := shellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		logger.Error("Failed to create system tray icon")
		return fmt.Errorf("failed to create tray icon")
	}

	logger.Info("System tray icon created successfully")
	return nil
}

// UpdateTrayIcon updates the tray icon based on connection status
func UpdateTrayIcon(connected bool) {
	logger.Debug("Updating tray icon - connected: %t", connected)

	var hIcon uintptr

	// Try to load custom icon first
	if connected {
		// For connected state, try to load custom icon
		hIcon, _, _ = loadImage.Call(
			0,
			uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ahcli.ico"))),
			1, 0, 0, LR_LOADFROMFILE,
		)
		if hIcon != 0 {
			logger.Debug("Using custom icon for connected state")
		}
	}

	// Fallback to system icons if custom icon not available
	if hIcon == 0 {
		hInstance, _, _ := getModuleHandle.Call(0)
		var iconID uintptr = 32513 // IDI_ERROR (disconnected - red-ish)
		if connected {
			iconID = 32516 // IDI_WINLOGO (connected - green-ish)
		}
		hIcon, _, _ = loadIcon.Call(hInstance, iconID)
		logger.Debug("Using system icon %d for connection state", iconID)
	}

	nid := NOTIFYICONDATA{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		Hwnd:   hwnd,
		UID:    trayIconID,
		UFlags: NIF_ICON | NIF_TIP,
		HIcon:  hIcon,
	}

	// Update tooltip
	tooltip := "AHCLI Voice Chat - Disconnected"
	if connected {
		tooltip = "AHCLI Voice Chat - Connected"
	}
	copy(nid.SzTip[:], syscall.StringToUTF16(tooltip))

	shellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	logger.Debug("Tray icon updated with tooltip: %s", tooltip)
}

// ShowTrayMenu shows the context menu when right-clicking the tray icon
func ShowTrayMenu() {
	logger.Debug("Showing tray context menu")

	// Create popup menu
	hMenu, _, _ := createPopupMenu.Call()
	if hMenu == 0 {
		logger.Error("Failed to create popup menu")
		return
	}
	defer destroyMenu.Call(hMenu)

	// Get current state for menu items
	state := appState.GetState()
	connected := state["connected"].(bool)
	currentChannel := state["currentChannel"]

	logger.Debug("Building menu - connected: %t, channel: %v", connected, currentChannel)

	// Menu items - keeping it minimal and purposeful
	menuItems := []struct {
		text string
		id   uintptr
	}{
		{"Open Voice Chat UI", 1001},
		{"", 0}, // Separator
	}

	// Add connection status (read-only)
	if connected {
		if currentChannel != nil && currentChannel.(string) != "" {
			menuItems = append(menuItems, struct {
				text string
				id   uintptr
			}{fmt.Sprintf("📡 Channel: %s", currentChannel), 0})
		}
		menuItems = append(menuItems, struct {
			text string
			id   uintptr
		}{"🟢 Connected", 0})
	} else {
		menuItems = append(menuItems, struct {
			text string
			id   uintptr
		}{"🔴 Disconnected", 0})
	}

	menuItems = append(menuItems, []struct {
		text string
		id   uintptr
	}{
		{"", 0}, // Separator
		{"Exit AHCLI", 1002},
	}...)

	// Add menu items
	for _, item := range menuItems {
		if item.text == "" {
			// Separator
			appendMenu.Call(hMenu, 0x800, 0, 0) // MF_SEPARATOR
		} else {
			flags := uintptr(0) // MF_STRING
			if item.id == 0 {
				flags = 0x1 // MF_GRAYED (disabled)
			}
			appendMenu.Call(hMenu, flags, item.id, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(item.text))))
		}
	}

	logger.Debug("Menu items added: %d total", len(menuItems))

	// Get cursor position
	var pt POINT
	getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// Make our window foreground so menu appears correctly
	setForegroundWindow.Call(hwnd)

	// Show menu and get selection
	cmd, _, _ := trackPopupMenu.Call(hMenu, TPM_RIGHTBUTTON|0x100, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)

	// Post a message to ourselves to close the menu cleanly
	postMessage.Call(hwnd, 0, 0, 0)

	logger.Debug("Menu command selected: %d", cmd)

	// Handle menu commands
	switch cmd {
	case 1001: // Open UI
		logger.Info("Tray menu: Opening Voice Chat UI")
		openVoiceChatUI()
	case 1002: // Exit
		logger.Info("Tray menu: Exiting application")
		exitApplication()
	default:
		if cmd != 0 {
			logger.Debug("Tray menu: Unknown command %d", cmd)
		}
	}
}

// openVoiceChatUI launches browser to the web interface
func openVoiceChatUI() {
	url := fmt.Sprintf("http://localhost:%d", webServerPort)

	logger.Info("Opening Voice Chat UI: %s", url)

	// Try Chrome app mode first (cleanest)
	browsers := [][]string{
		{"chrome", "--app=" + url, "--disable-web-security", "--disable-features=TranslateUI"},
		{"msedge", "--app=" + url, "--disable-web-security"},
		{"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe", "--app=" + url},
		{"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe", "--app=" + url},
	}

	for i, browser := range browsers {
		logger.Debug("Trying browser %d: %s", i+1, browser[0])
		cmd := exec.Command(browser[0], browser[1:]...)
		if err := cmd.Start(); err == nil {
			logger.Info("Successfully launched browser: %s", browser[0])
			appState.AddMessage("Voice Chat UI opened", "info")
			return
		} else {
			logger.Debug("Browser %s failed: %v", browser[0], err)
		}
	}

	// Fallback to default browser
	logger.Debug("All specific browsers failed, trying default browser")
	logger.Info("Opening in default browser...")
	err := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	if err != nil {
		logger.Error("Failed to open default browser: %v", err)
		appState.AddMessage("Failed to open Voice Chat UI", "error")
	} else {
		appState.AddMessage("Voice Chat UI opened in default browser", "info")
	}
}

// exitApplication performs graceful shutdown
func exitApplication() {
	logger.Info("Exit requested from system tray")
	appState.AddMessage("AHCLI shutting down...", "info")

	// Remove tray icon
	nid := NOTIFYICONDATA{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		Hwnd:   hwnd,
		UID:    trayIconID,
	}

	ret, _, _ := shellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
	if ret != 0 {
		logger.Debug("Tray icon removed successfully")
	} else {
		logger.Error("Failed to remove tray icon")
	}

	logger.Info("AHCLI shutdown complete")
	syscall.Exit(0)
}

// HandleTrayMessage processes tray icon messages
func HandleTrayMessage(msg uintptr) {
	switch msg {
	case WM_RBUTTONUP:
		logger.Debug("Tray icon right-clicked")
		ShowTrayMenu()
	case WM_LBUTTONUP:
		logger.Debug("Tray icon left-clicked")
		// Single click - open UI
		openVoiceChatUI()
	default:
		logger.Debug("Unknown tray message: %d", msg)
	}
}

// CleanupTray removes the tray icon (called on shutdown)
func CleanupTray() {
	logger.Debug("Cleaning up system tray icon")

	nid := NOTIFYICONDATA{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		Hwnd:   hwnd,
		UID:    trayIconID,
	}

	ret, _, _ := shellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
	if ret != 0 {
		logger.Info("System tray icon removed successfully")
	} else {
		logger.Error("Failed to remove system tray icon during cleanup")
	}
}
