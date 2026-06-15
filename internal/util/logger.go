package util

import (
	"fmt"
	"os"
	"time"
)

// ANSI colors (work in modern Windows Terminal, PowerShell, and most Unix terminals).
const (
	reset   = "\033[0m"
	yellow  = "\033[93m"
	magenta = "\033[95m"
	green   = "\033[92m"
	red     = "\033[91m"
)

// Logger prints colored tagged log lines to stdout.
type Logger struct {
	id     string
	colors bool
}

func NewLogger(id string) *Logger {
	return &Logger{id: id, colors: supportsColor()}
}

func (l *Logger) SetID(id string) { l.id = id }

func (l *Logger) tag() string {
	if l.id == "" {
		return "-"
	}
	return l.id
}

func (l *Logger) paint(color, text string) string {
	if !l.colors {
		return text
	}
	return color + text + reset
}

func (l *Logger) Info(msg string) {
	now := time.Now().Format("15:04:05")
	fmt.Printf("%s %s: %s\n",
		l.paint(yellow, now),
		l.paint(magenta, "["+l.tag()+"]"),
		l.paint(green, msg),
	)
}

func (l *Logger) Danger(msg string) {
	now := time.Now().Format("15:04:05")
	fmt.Printf("%s %s: %s\n",
		l.paint(yellow, now),
		l.paint(magenta, "["+l.tag()+"]"),
		l.paint(red, msg),
	)
}

// supportsColor returns false when output is redirected to a file/pipe.
func supportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Windows 10+ and most Unix terminals understand ANSI when writing to a TTY.
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
