package jdwp

import (
	"context"
	"errors"
)

var (
	ErrUsage       = errors.New("usage")
	ErrToolMissing = errors.New("tool missing")
	ErrNoProcess   = errors.New("no process")
	ErrProbe       = errors.New("probe failed")
)

type Session struct {
	Package  string
	Serial   string
	PID      int
	Port     int
	Activity string // empty when launch used monkey
}

// Client attaches and resets a JDWP forward. Temporary: move to the
// consumer (cli) when that shape stabilizes. Fake shared across modules
// justifies keeping it here for the MVP.
type Client interface {
	Attach(ctx context.Context, serial, pkg string, port int) (Session, error)
	Reset(ctx context.Context, serial, pkg string, port int) error
}
