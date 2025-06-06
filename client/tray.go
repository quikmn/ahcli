//go:build windows

// FILE: client/tray.go

package main

import (
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

	// Get module handle for icon
	hInstance, _, _ := getModuleHandle.Call(0)

	// Load default application icon
	hIcon, _, _ := loadIcon.Call(hInstance, uintptr(32512)) // IDI_APPLICATION

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
		return fmt.Errorf("failed to create tray icon")
	}

	LogInfo("System tray icon created successfully")
	return nil
}

// UpdateTrayIcon updates the tray icon based on connection status
func UpdateTrayIcon(connected bool) {
	// Get module handle
	hInstance, _, _ := getModuleHandle.Call(0)

	// Use different icons based on connection status
	var iconID uintptr = 32512 // IDI_APPLICATION (default)
	if connected {
		iconID = 32516 // IDI_WINLOGO (connected - green-ish)
	} else {
		iconID = 32513 // IDI_ERROR (disconnected - red-ish)
	}

	hIcon, _, _ := loadIcon.Call(hInstance, iconID)

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
}

// ShowTrayMenu shows the context menu when right-clicking the tray icon
func ShowTrayMenu() {
	// Create popup menu
	hMenu, _, _ := createPopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer destroyMenu.Call(hMenu)

	// Get current state for menu items
	state := appState.GetState()
	connected := state["connected"].(bool)
	currentChannel := state["currentChannel"]

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

	// Get cursor position
	var pt POINT
	getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// Make our window foreground so menu appears correctly
	setForegroundWindow.Call(hwnd)

	// Show menu and get selection
	cmd, _, _ := trackPopupMenu.Call(hMenu, TPM_RIGHTBUTTON|0x100, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)

	// Post a message to ourselves to close the menu cleanly
	postMessage.Call(hwnd, 0, 0, 0)

	// Handle menu commands
	switch cmd {
	case 1001: // Open UI
		LogInfo("Menu: Opening Voice Chat UI")
		openVoiceChatUI()
	case 1002: // Exit
		LogInfo("Menu: Exiting application")
		exitApplication()
	default:
		if cmd != 0 {
			LogInfo("Menu: Unknown command %d", cmd)
		}
	}
}

// openVoiceChatUI launches browser to the web interface
func openVoiceChatUI() {
	url := fmt.Sprintf("http://localhost:%d", webServerPort)

	LogInfo("Opening Voice Chat UI: %s", url)

	// Try Chrome app mode first (cleanest)
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

			// Update both systems with UI open message
			appState.AddMessage("Voice Chat UI opened", "info")
			WebTUIAddMessage("Voice Chat UI opened", "info")
			return
		}
	}

	// Fallback to default browser
	LogInfo("Opening in default browser...")
	exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()

	appState.AddMessage("Voice Chat UI opened in default browser", "info")
	WebTUIAddMessage("Voice Chat UI opened in default browser", "info")
}

// exitApplication performs graceful shutdown
func exitApplication() {
	LogInfo("Exit requested from system tray")

	// Update both systems with exit message
	appState.AddMessage("AHCLI shutting down...", "info")
	WebTUIAddMessage("AHCLI shutting down...", "info")

	// Remove tray icon
	nid := NOTIFYICONDATA{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		Hwnd:   hwnd,
		UID:    trayIconID,
	}
	shellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))

	// TODO: Add graceful cleanup here
	// - Close audio streams
	// - Disconnect from server
	// - Close web server

	LogInfo("AHCLI shutdown complete")

	// Exit cleanly
	syscall.Exit(0)
}

// HandleTrayMessage processes tray icon messages
func HandleTrayMessage(msg uintptr) {
	switch msg {
	case WM_RBUTTONUP:
		ShowTrayMenu()
	case WM_LBUTTONUP:
		// Double-click or single click - open UI
		openVoiceChatUI()
	}
}

// CleanupTray removes the tray icon (called on shutdown)
func CleanupTray() {
	nid := NOTIFYICONDATA{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		Hwnd:   hwnd,
		UID:    trayIconID,
	}
	shellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
	LogInfo("System tray icon removed")
}
