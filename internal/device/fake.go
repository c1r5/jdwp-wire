package device

import (
	"context"
	"fmt"
)

type Fake struct {
	Devices       []Device
	Procs         map[string][]Process
	Apps          []App
	DebugPackages map[string]bool
	Missing       map[string]bool
	ListErr       error
	AppsErr       error
	DebugErr      error
}

func (f *Fake) List(context.Context) ([]Device, error) {
	if f.ListErr != nil {
		return nil, fmt.Errorf("device: list: %w", f.ListErr)
	}
	out := make([]Device, len(f.Devices))
	copy(out, f.Devices)
	return out, nil
}

func (f *Fake) Resolve(ctx context.Context, serial string) (Device, error) {
	devs, err := f.List(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("device: resolve: %w", err)
	}
	d, err := resolve(devs, serial)
	if err != nil {
		return Device{}, fmt.Errorf("device: resolve: %w", err)
	}
	return d, nil
}

func (f *Fake) ListApps(_ context.Context, serial string, includeSystem bool) ([]App, error) {
	if serial == "" {
		return nil, fmt.Errorf("device: apps: %w", ErrUsage)
	}
	if f.AppsErr != nil {
		return nil, fmt.Errorf("device: apps: %w", f.AppsErr)
	}
	out := make([]App, 0, len(f.Apps))
	for _, app := range f.Apps {
		if app.System && !includeSystem {
			continue
		}
		out = append(out, app)
	}
	return out, nil
}

func (f *Fake) Debuggable(_ context.Context, serial, pkg string) (bool, error) {
	if serial == "" || pkg == "" {
		return false, fmt.Errorf("device: debuggable: %w", ErrUsage)
	}
	if f.DebugErr != nil {
		return false, fmt.Errorf("device: debuggable: %w", f.DebugErr)
	}
	if f.Missing[pkg] {
		return false, fmt.Errorf("device: debuggable: %w", ErrPackageNotFound)
	}
	return f.DebugPackages[pkg], nil
}

func (f *Fake) Pidof(_ context.Context, serial, pkg string) ([]Process, error) {
	if serial == "" || pkg == "" {
		return nil, fmt.Errorf("device: pidof: %w", ErrUsage)
	}
	if f.Procs == nil {
		return nil, nil
	}
	src := f.Procs[pkg]
	out := make([]Process, len(src))
	copy(out, src)
	return out, nil
}
