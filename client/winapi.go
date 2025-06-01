//go:build windows

// FILE: client/winapi.go

package main

import (
	"syscall"
)

// Consolidated Windows API declarations - used by ptt.go, tray.go, and main.go
var (
	// DLLs
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	
	// User32 functions
	procGetKeyState      = user32.NewProc("GetAsyncKeyState")
	createWindowEx       = user32.NewProc("CreateWindowExW")
	defWindowProc        = user32.NewProc("DefWindowProcW")
	getMessage           = user32.NewProc("GetMessageW")
	translateMessage     = user32.NewProc("TranslateMessage")
	dispatchMessage      = user32.NewProc("DispatchMessageW")
	registerClassEx      = user32.NewProc("RegisterClassExW")
	createPopupMenu      = user32.NewProc("CreatePopupMenu")
	appendMenu           = user32.NewProc("AppendMenuW")
	trackPopupMenu       = user32.NewProc("TrackPopupMenu")
	destroyMenu          = user32.NewProc("DestroyMenu")
	getCursorPos         = user32.NewProc("GetCursorPos")
	setForegroundWindow  = user32.NewProc("SetForegroundWindow")
	postMessage          = user32.NewProc("PostMessageW")
	loadIcon             = user32.NewProc("LoadIconW")
	
	// Shell32 functions
	shellNotifyIcon      = shell32.NewProc("Shell_NotifyIconW")
	
	// Kernel32 functions
	getModuleHandle      = kernel32.NewProc("GetModuleHandleW")
)

// Windows constants
const (
	// Messages
	WM_USER       = 0x0400
	WM_TRAYICON   = WM_USER + 1
	WM_LBUTTONUP  = 0x0202
	WM_RBUTTONUP  = 0x0205
	
	// Tray icon operations
	NIM_ADD       = 0
	NIM_MODIFY    = 1
	NIM_DELETE    = 2
	
	// Tray icon flags
	NIF_MESSAGE   = 1
	NIF_ICON      = 2
	NIF_TIP       = 4
	
	// Menu flags
	TPM_RIGHTBUTTON = 2
)

// Windows structures
type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type NOTIFYICONDATA struct {
	CbSize           uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type POINT struct {
	X, Y int32
}

// Shared variables
var (
	hwnd        uintptr
	trayIconID  uint32 = 1
)

// PTT helper functions (moved from ptt.go)
func isKeyDown(vk uint16) bool {
	ret, _, _ := procGetKeyState.Call(uintptr(vk))
	return (ret & 0x8000) != 0
}