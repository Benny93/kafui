package ui

import (
	"fmt"
	"os"
	"time"
)

func debugLog(format string, args ...interface{}) {
	// Open log file in append mode
	f, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Add timestamp to log
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logMsg := fmt.Sprintf("%s: %s\n", timestamp, fmt.Sprintf(format, args...))

	f.WriteString(logMsg)
}
