// client/logger.go
package main

import (
	"flag"
	"log"
	"os"
)

var (
	debugMode   bool
	debugLogger *log.Logger
	logFile     *os.File
)

func InitLogger() {
	// Parse command line flags
	flag.BoolVar(&debugMode, "debug", false, "Enable debug logging to client.log")
	flag.Parse()

	if !debugMode {
		return // No logging if debug flag not set
	}

	// Open log file for writing
	var err error
	logFile, err = os.OpenFile("client.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		println("Failed to open log file:", err.Error())
		return
	}

	debugLogger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	LogInfo("=== AHCLI Client Debug Session Started ===")
}

func CloseLogger() {
	if logFile != nil {
		LogInfo("=== AHCLI Client Debug Session Ended ===")
		logFile.Close()
	}
}

// Logging functions that only work when --debug flag is used
func LogInfo(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[INFO] "+format, args...)
	}
}

func LogError(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[ERROR] "+format, args...)
	}
}

func LogDebug(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[DEBUG] "+format, args...)
	}
}

func LogPTT(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[PTT] "+format, args...)
	}
}

func LogAudio(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[AUDIO] "+format, args...)
	}
}

func LogNet(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[NET] "+format, args...)
	}
}

func LogSend(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[SEND] "+format, args...)
	}
}

func LogPlayback(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[PLAYBACK] "+format, args...)
	}
}

func LogTest(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[TEST] "+format, args...)
	}
}

func LogMain(format string, args ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[MAIN] "+format, args...)
	}
}