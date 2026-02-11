package logger

import (
	"fmt"
	"os"
)

// Global configuration
var (
	Enabled bool
	LogPath string
)

// Debugf logs a message to stderr (and optionally a file) if debug logging is enabled
func Debugf(format string, args ...any) {
	if !Enabled {
		return
	}

	msg := fmt.Sprintf("[DEBUG] "+format+"\n", args...)

	// Always print to stderr
	fmt.Fprint(os.Stderr, msg)

	// Optionally log to file
	if LogPath != "" {
		f, err := os.OpenFile(LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] failed to open log file: %v\n", err)
			return
		}
		defer f.Close()

		if _, err := f.WriteString(msg); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] failed to write to log file: %v\n", err)
		}
	}
}
