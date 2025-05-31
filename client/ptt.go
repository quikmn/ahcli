//go:build windows

package main

import (
	"fmt"
	"sync"
	"syscall"
	"time"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procGetKeyState   = user32.NewProc("GetAsyncKeyState")

	isPressedMu sync.RWMutex
	isPressed   bool
	pttKeyCode  uint16 = 0xA0 // VK_LSHIFT, change to F1 = 0x70, Space = 0x20, etc.
)

func keyNameToVKCode(key string) uint16 {
	switch key {
	case "LSHIFT":
		return 0xA0
	case "RSHIFT":
		return 0xA1
	case "LCTRL":
		return 0xA2
	case "RCTRL":
		return 0xA3
	case "LALT":
		return 0xA4
	case "RALT":
		return 0xA5
	case "SPACE":
		return 0x20
	case "TAB":
		return 0x09
	case "ENTER":
		return 0x0D
	case "ESC":
		return 0x1B
	case "BACKSPACE":
		return 0x08
	case "CAPSLOCK":
		return 0x14
	case "INSERT":
		return 0x2D
	case "DELETE":
		return 0x2E
	case "HOME":
		return 0x24
	case "END":
		return 0x23
	case "PAGEUP":
		return 0x21
	case "PAGEDOWN":
		return 0x22
	case "UP":
		return 0x26
	case "DOWN":
		return 0x28
	case "LEFT":
		return 0x25
	case "RIGHT":
		return 0x27
	case "F1":
		return 0x70
	case "F2":
		return 0x71
	case "F3":
		return 0x72
	case "F4":
		return 0x73
	case "F5":
		return 0x74
	case "F6":
		return 0x75
	case "F7":
		return 0x76
	case "F8":
		return 0x77
	case "F9":
		return 0x78
	case "F10":
		return 0x79
	case "F11":
		return 0x7A
	case "F12":
		return 0x7B
	case "F13":
		return 0x7C
	case "F14":
		return 0x7D
	case "F15":
		return 0x7E
	case "F16":
		return 0x7F
	case "F17":
		return 0x80
	case "F18":
		return 0x81
	case "F19":
		return 0x82
	case "F20":
		return 0x83
	case "F21":
		return 0x84
	case "F22":
		return 0x85
	case "F23":
		return 0x86
	case "F24":
		return 0x87
	case "NUMLOCK":
		return 0x90
	case "SCROLLLOCK":
		return 0x91
	case "PRINTSCREEN":
		return 0x2C
	case "PAUSE":
		return 0x13
	case "NUM0":
		return 0x60
	case "NUM1":
		return 0x61
	case "NUM2":
		return 0x62
	case "NUM3":
		return 0x63
	case "NUM4":
		return 0x64
	case "NUM5":
		return 0x65
	case "NUM6":
		return 0x66
	case "NUM7":
		return 0x67
	case "NUM8":
		return 0x68
	case "NUM9":
		return 0x69
	case "A":
		return 0x41
	case "B":
		return 0x42
	case "C":
		return 0x43
	case "D":
		return 0x44
	case "E":
		return 0x45
	case "F":
		return 0x46
	case "G":
		return 0x47
	case "H":
		return 0x48
	case "I":
		return 0x49
	case "J":
		return 0x4A
	case "K":
		return 0x4B
	case "L":
		return 0x4C
	case "M":
		return 0x4D
	case "N":
		return 0x4E
	case "O":
		return 0x4F
	case "P":
		return 0x50
	case "Q":
		return 0x51
	case "R":
		return 0x52
	case "S":
		return 0x53
	case "T":
		return 0x54
	case "U":
		return 0x55
	case "V":
		return 0x56
	case "W":
		return 0x57
	case "X":
		return 0x58
	case "Y":
		return 0x59
	case "Z":
		return 0x5A
	case "0":
		return 0x30
	case "1":
		return 0x31
	case "2":
		return 0x32
	case "3":
		return 0x33
	case "4":
		return 0x34
	case "5":
		return 0x35
	case "6":
		return 0x36
	case "7":
		return 0x37
	case "8":
		return 0x38
	case "9":
		return 0x39
	default:
		return 0
	}
}

// StartPTTListener starts polling the PTT key state.
func StartPTTListener() {
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			pressed := isKeyDown(pttKeyCode)
			fmt.Println("[PTT Debug] Polling... Pressed =", pressed)

			isPressedMu.Lock()
			isPressed = pressed
			isPressedMu.Unlock()
		}
	}()
}

// IsPTTActive returns whether the PTT key is currently being held.
func IsPTTActive() bool {
	isPressedMu.RLock()
	defer isPressedMu.RUnlock()
	return isPressed
}

// isKeyDown returns true if the given virtual key is down.
func isKeyDown(vk uint16) bool {
	ret, _, _ := procGetKeyState.Call(uintptr(vk))
	return (ret & 0x8000) != 0
}
