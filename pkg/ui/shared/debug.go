package shared

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	// Maximum number of lines to keep in the log file
	maxLogLines = 10000
	// Number of lines to keep when rotating (keep recent 80%)
	keepLogLines = 8000
	logFileName  = "debug.log"
)

var (
	logMutex       sync.Mutex
	logInitialized bool
)

// InitDebugLog initializes the debug logging by cleaning up any existing log file
func InitDebugLog() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logInitialized {
		return
	}

	// Delete existing log file on startup
	if _, err := os.Stat(logFileName); err == nil {
		os.Remove(logFileName)
	}

	logInitialized = true
}

// rotateLogIfNeeded checks if the log file is too large and rotates it
func rotateLogIfNeeded() error {
	// Check if file exists and get line count
	file, err := os.Open(logFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, no rotation needed
		}
		return err
	}

	// Count lines in the file
	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	// If file is not too large, no rotation needed
	if lineCount <= maxLogLines {
		return nil
	}

	// Read all lines
	file, err = os.Open(logFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Keep only the most recent lines
	startIndex := len(lines) - keepLogLines
	if startIndex < 0 {
		startIndex = 0
	}

	// Write back the truncated log
	file, err = os.Create(logFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for i := startIndex; i < len(lines); i++ {
		if _, err := writer.WriteString(lines[i] + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func DebugLog(format string, args ...interface{}) {
	// Try to acquire lock with timeout to prevent deadlock
	if !tryLockWithTimeout() {
		return // Skip logging if we can't acquire lock
	}
	defer logMutex.Unlock()

	// Initialize log on first use if not already initialized
	if !logInitialized {
		// Don't call InitDebugLog here as it would cause deadlock
		// Just set the flag and continue
		logInitialized = true
	}

	// Rotate log if needed (check every write to keep it manageable)
	if err := rotateLogIfNeeded(); err != nil {
		// If rotation fails, continue with logging but ignore the error
		// to prevent breaking the application
	}

	// Open log file in append mode
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Add timestamp to log
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logMsg := fmt.Sprintf("%s: %s\n", timestamp, fmt.Sprintf(format, args...))

	f.WriteString(logMsg)
}

// tryLockWithTimeout attempts to acquire the log mutex with a timeout
func tryLockWithTimeout() bool {
	done := make(chan bool, 1)
	go func() {
		logMutex.Lock()
		done <- true
	}()
	
	select {
	case <-done:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}
