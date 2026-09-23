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
	ErrPackageNotFound = errors.New("package not installed")
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

// App is an installed package. Label is empty when the device did not
// print a non-localized name. PID is 0 when nothing is running.
// System is a system package. ListApps omits these unless includeSystem is set.
// The adb client applies that cut with `pm list packages -3` and leaves System false.
type App struct {
	Package string
	Label   string
	PID     int
	System  bool
}

// Client lists devices and processes. Temporary: move to the consumer
// (cli/jdwp) when that shape stabilizes. Fake shared across modules
// justifies keeping it here for the MVP.
type Client interface {
	List(ctx context.Context) ([]Device, error)
	Resolve(ctx context.Context, serial string) (Device, error)
	Pidof(ctx context.Context, serial, pkg string) ([]Process, error)
	// ListApps returns installed packages. includeSystem adds system packages;
	// otherwise the result is third-party packages only.
	ListApps(ctx context.Context, serial string, includeSystem bool) ([]App, error)
	Debuggable(ctx context.Context, serial, pkg string) (bool, error)
}
