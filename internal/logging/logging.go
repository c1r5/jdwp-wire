// Package logging writes one human step per call.
// The colored logging library is imported only here, so replacing it stays in this package.
package logging

import (
	"io"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

const timeFormat = "15:04:05"

// Logger writes step lines to w. Color follows w: a terminal is colored, a pipe or buffer is not.
type Logger struct {
	mu     sync.Mutex
	raw    *log.Logger
	styles *log.Styles
}

// New returns a logger that writes to w. A nil writer discards output.
func New(w io.Writer) *Logger {
	if w == nil {
		w = io.Discard
	}
	styles := log.DefaultStyles()
	styles.Timestamp = lipgloss.NewStyle().Faint(true)
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("[ok]").
		Bold(true).
		Foreground(lipgloss.Color("42"))
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("[skip]").
		Bold(true).
		Foreground(lipgloss.Color("220"))
	raw := log.NewWithOptions(w, log.Options{
		ReportTimestamp: true,
		TimeFormat:      timeFormat,
		Level:           log.DebugLevel,
	})
	raw.SetStyles(styles)
	return &Logger{raw: raw, styles: styles}
}

// OK records a finished step: HH:MM:SS [ok] [module] action.
func (l *Logger) OK(module, action string) {
	l.step(log.InfoLevel, module, action)
}

// Skip records a step that did not need to run: HH:MM:SS [skip] [module] action.
func (l *Logger) Skip(module, action string) {
	l.step(log.WarnLevel, module, action)
}

func (l *Logger) step(level log.Level, module, action string) {
	st := *l.styles
	st.Message = lipgloss.NewStyle().Foreground(moduleColor(module))
	l.mu.Lock()
	l.raw.SetStyles(&st)
	l.raw.Log(level, "["+module+"] "+action)
	l.mu.Unlock()
}
