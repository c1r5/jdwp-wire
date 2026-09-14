package device

import (
	"context"
	"errors"
)

var (
	ErrNoDevice        = errors.New("no device")
	ErrAmbiguousDevice = errors.New("ambiguous device")
	ErrDeviceNotFound  = errors.New("device not found")
	ErrDeviceUnusable  = errors.New("device unusable")
	ErrToolMissing     = errors.New("tool missing")
	ErrUsage           = errors.New("usage")
)

type State string

const (
	StateDevice       State = "device"
	StateOffline      State = "offline"
	StateUnauthorized State = "unauthorized"
	StateUnknown      State = "unknown"
)

type Kind string

const (
	KindUSB      Kind = "usb"
	KindEmulator Kind = "emulator"
	KindUnknown  Kind = "unknown"
)

type Device struct {
	Serial string
	State  State
	Kind   Kind
	Model  string // product/model from `adb devices -l`; may be empty
}

func (d Device) Usable() bool { return d.State == StateDevice }

type Process struct {
	PID     int
	Package string // "com.alvo" or "com.alvo:remote"
}

// Client lists devices and processes. Temporary: move to the consumer
// (cli/jdwp) when that shape stabilizes. Fake shared across modules
// justifies keeping it here for the MVP.
type Client interface {
	List(ctx context.Context) ([]Device, error)
	Resolve(ctx context.Context, serial string) (Device, error)
	Pidof(ctx context.Context, serial, pkg string) ([]Process, error)
}
