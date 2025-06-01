// server/logger.go
package main

import (
	"log"
	"os"
)

var (
	logger *log.Logger
)

func InitLogger() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	LogInfo("=== AHCLI Server Started ===")
}

func CloseLogger() {
	LogInfo("=== AHCLI Server Ended ===")
}

// Unified logging functions - always output to console
func LogInfo(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[INFO] "+format, args...)
	}
}

func LogError(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[ERROR] "+format, args...)
	}
}

func LogDebug(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

func LogClient(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[CLIENT] "+format, args...)
	}
}

func LogAudio(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[AUDIO] "+format, args...)
	}
}

func LogNet(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf("[NET] "+format, args...)
	}
}