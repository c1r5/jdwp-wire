package frida

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogPath is the daily file .jdt/<pkg>/frida/AAAA-MM-DD.log.
func LogPath(dir string, now time.Time) string {
	return filepath.Join(dir, now.Format("2006-01-02")+".log")
}

// Log appends rewritten lines to the daily file.
type Log struct {
	path string
	f    *os.File
}

// OpenLog creates dir and opens the daily file for append.
func OpenLog(dir string, now time.Time) (*Log, error) {
	return OpenLogAt(LogPath(dir, now))
}

// OpenLogAt appends to path, creating the parent directory.
func OpenLogAt(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("frida: log: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("frida: log: %w", err)
	}
	return &Log{path: path, f: f}, nil
}

// Path is the absolute or joined path passed to OpenLog.
func (l *Log) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// WriteLine appends one rewritten line and a newline.
func (l *Log) WriteLine(s string) error {
	if l == nil || l.f == nil {
		return fmt.Errorf("frida: log: closed")
	}
	if _, err := io.WriteString(l.f, s+"\n"); err != nil {
		return fmt.Errorf("frida: log: %w", err)
	}
	return nil
}

// Close flushes the file.
func (l *Log) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	if err != nil {
		return fmt.Errorf("frida: log: %w", err)
	}
	return nil
}
