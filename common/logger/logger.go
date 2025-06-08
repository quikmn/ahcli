// FILE: common/logger/logger.go
package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Log levels
const (
	FATAL = iota
	ERROR
	WARN
	INFO
	DEBUG
)

var (
	// Global logger instance
	globalLogger *Logger
	initOnce     sync.Once

	// Elite performance cache for filename -> component mapping
	componentCache = sync.Map{}
)

// Logger manages unified logging across the application
type Logger struct {
	mu sync.RWMutex

	appName    string
	logFile    *os.File
	fileLogger *log.Logger
	debugMode  bool

	// Console colors
	colors map[int]string
}

// Init initializes the global logger for the application
func Init(appName string) error {
	var initErr error

	initOnce.Do(func() {
		globalLogger = &Logger{
			appName: appName,
			colors: map[int]string{
				FATAL: "\033[1;31m", // Bright red
				ERROR: "\033[0;31m", // Red
				WARN:  "\033[0;33m", // Yellow
				INFO:  "\033[0;32m", // Green
				DEBUG: "\033[0;37m", // Gray
			},
		}

		// Create log file
		logFileName := fmt.Sprintf("ahcli-%s.log", appName)
		file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			initErr = fmt.Errorf("failed to open log file %s: %v", logFileName, err)
			return
		}

		globalLogger.logFile = file
		globalLogger.fileLogger = log.New(file, "", 0) // Custom formatting

		// Log startup - use manual component for init
		globalLogger.logToFile(INFO, "SYSTEM", fmt.Sprintf("=== AHCLI %s Started ===", appName))
	})

	return initErr
}

// SetDebugMode enables or disables debug logging
func SetDebugMode(enabled bool) {
	if globalLogger != nil {
		globalLogger.mu.Lock()
		globalLogger.debugMode = enabled
		globalLogger.mu.Unlock()
	}
}

// GetLogPath returns the current log file path
func GetLogPath() string {
	if globalLogger != nil && globalLogger.logFile != nil {
		return globalLogger.logFile.Name()
	}
	return ""
}

// Elite auto-context logging functions - zero manual typing required
func Fatal(format string, args ...interface{}) {
	component := getComponent()
	logWithLevel(FATAL, component, format, args...)
	os.Exit(1)
}

func Error(format string, args ...interface{}) {
	component := getComponent()
	logWithLevel(ERROR, component, format, args...)
}

func Warn(format string, args ...interface{}) {
	component := getComponent()
	logWithLevel(WARN, component, format, args...)
}

func Info(format string, args ...interface{}) {
	component := getComponent()
	logWithLevel(INFO, component, format, args...)
}

func Debug(format string, args ...interface{}) {
	component := getComponent()
	logWithLevel(DEBUG, component, format, args...)
}

// Elite runtime call stack detection with performance caching
func getComponent() string {
	_, file, _, ok := runtime.Caller(2) // Skip logger.Info() and logWithLevel()
	if !ok {
		return "UNKNOWN"
	}

	// Check cache first for elite performance
	if cached, exists := componentCache.Load(file); exists {
		return cached.(string)
	}

	// Smart filename-to-component mapping
	component := mapFileToComponent(file)
	componentCache.Store(file, component) // Cache for next time
	return component
}

// Senior engineer filename mapping logic
func mapFileToComponent(file string) string {
	// Convert to lowercase for case-insensitive matching
	file = strings.ToLower(file)

	switch {
	case strings.Contains(file, "audio"):
		return "AUDIO"
	case strings.Contains(file, "webserver"):
		return "INTERFACE"
	case strings.Contains(file, "tray"):
		return "INTERFACE"
	case strings.Contains(file, "net"):
		return "NET"
	case strings.Contains(file, "crypto"):
		return "CRYPTO"
	case strings.Contains(file, "chat"):
		return "CHAT"
	case strings.Contains(file, "server/main"):
		return "SERVER"
	case strings.Contains(file, "server/"):
		return "SERVER"
	case strings.Contains(file, "client/main"):
		return "CLIENT"
	case strings.Contains(file, "config"):
		return "CONFIG"
	case strings.Contains(file, "appstate"):
		return "STATE"
	default:
		return "GENERAL"
	}
}

// logWithLevel handles the actual logging logic
func logWithLevel(level int, component, format string, args ...interface{}) {
	if globalLogger == nil {
		// Fallback to console if logger not initialized
		fmt.Printf("[UNINITIALIZED] "+format+"\n", args...)
		return
	}

	// Skip debug messages unless debug mode is enabled
	if level == DEBUG {
		globalLogger.mu.RLock()
		debugEnabled := globalLogger.debugMode
		globalLogger.mu.RUnlock()

		if !debugEnabled {
			return
		}
	}

	message := fmt.Sprintf(format, args...)

	// Always log to file
	globalLogger.logToFile(level, component, message)

	// Log to console for important messages (INFO and above)
	if level <= INFO {
		globalLogger.logToConsole(level, component, message)
	}
}

// logToFile writes structured logs to the file
func (l *Logger) logToFile(level int, component, message string) {
	if l.fileLogger == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Get timestamp and level
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := getLevelString(level)

	// Elite format: 2025-01-08 15:04:05.123 [INFO ] [AUDIO] message
	var logLine string
	if component != "" {
		logLine = fmt.Sprintf("%s [%-5s] [%s] %s", timestamp, levelStr, component, message)
	} else {
		logLine = fmt.Sprintf("%s [%-5s] %s", timestamp, levelStr, message)
	}

	l.fileLogger.Println(logLine)
}

// logToConsole writes colored logs to the console
func (l *Logger) logToConsole(level int, component, message string) {
	timestamp := time.Now().Format("15:04:05")
	levelStr := getLevelString(level)

	// Get color for this level
	color := l.colors[level]
	reset := "\033[0m"

	// Elite format: [15:04:05] INFO  [AUDIO] message
	if component != "" {
		fmt.Printf("%s[%s] %-5s [%s] %s%s\n", color, timestamp, levelStr, component, message, reset)
	} else {
		fmt.Printf("%s[%s] %-5s %s%s\n", color, timestamp, levelStr, message, reset)
	}
}

// getLevelString returns the string representation of a log level
func getLevelString(level int) string {
	switch level {
	case FATAL:
		return "FATAL"
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// Close closes the log file (call during shutdown)
func Close() {
	if globalLogger != nil && globalLogger.logFile != nil {
		globalLogger.logToFile(INFO, "SYSTEM", fmt.Sprintf("=== AHCLI %s Ended ===", globalLogger.appName))
		globalLogger.logFile.Close()
	}
}

// Rotate rotates the current log file (for future log rotation feature)
func Rotate() error {
	if globalLogger == nil || globalLogger.logFile == nil {
		return fmt.Errorf("logger not initialized")
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	// Close current file
	oldFileName := globalLogger.logFile.Name()
	globalLogger.logFile.Close()

	// Rename current log with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s", oldFileName, timestamp)
	os.Rename(oldFileName, backupName)

	// Create new log file
	newFile, err := os.OpenFile(oldFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}

	globalLogger.logFile = newFile
	globalLogger.fileLogger = log.New(newFile, "", 0)

	globalLogger.logToFile(INFO, "SYSTEM", "Log file rotated")
	return nil
}
