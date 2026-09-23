package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

// appLabelTimeout bounds dumpsys package. A full dump is often slow, and
// resource labels are usually absent; the package list must still return.
const appLabelTimeout = 8 * time.Second

type ADB struct {
	r execx.Runner
}

func NewADB(r execx.Runner) *ADB {
	return &ADB{r: r}
}

func (a *ADB) List(ctx context.Context) ([]Device, error) {
	res, err := a.r.Run(ctx, "adb", "devices", "-l")
	if errors.Is(err, execx.ErrNotFound) {
		return nil, fmt.Errorf("device: list: %w", fmt.Errorf("%w: adb", ErrToolMissing))
	}
	if errors.Is(err, execx.ErrExit) {
		res, err = a.r.Run(ctx, "adb", "devices")
		if err != nil {
			return nil, wrapADB("list", err)
		}
		return parseDevices(res.Stdout), nil
	}
	if err != nil {
		return nil, wrapADB("list", err)
	}
	return parseDevices(res.Stdout), nil
}

func (a *ADB) Resolve(ctx context.Context, serial string) (Device, error) {
	devs, err := a.List(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("device: resolve: %w", err)
	}
	d, err := resolve(devs, serial)
	if err != nil {
		return Device{}, fmt.Errorf("device: resolve: %w", err)
	}
	return d, nil
}

func (a *ADB) ListApps(ctx context.Context, serial string) ([]App, error) {
	if serial == "" {
		return nil, fmt.Errorf("device: apps: %w", ErrUsage)
	}
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "pm", "list", "packages")
	if err != nil {
		return nil, wrapADB("apps", err)
	}
	procs, err := a.psAll(ctx, serial)
	if err != nil {
		return nil, err
	}
	labels := a.appLabels(ctx, serial)
	pkgs := parsePackageList(res.Stdout)
	out := make([]App, 0, len(pkgs))
	for _, pkg := range pkgs {
		out = append(out, App{
			Package: pkg,
			Label:   labels[pkg],
			PID:     pidForPackage(pkg, procs),
		})
	}
	return out, nil
}

func (a *ADB) appLabels(ctx context.Context, serial string) map[string]string {
	labelCtx, cancel := context.WithTimeout(ctx, appLabelTimeout)
	defer cancel()
	res, err := a.r.Run(labelCtx, "adb", "-s", serial, "shell", "dumpsys", "package")
	if err != nil && res.Stdout == "" {
		return nil
	}
	return parseAppLabels(res.Stdout)
}

func (a *ADB) psAll(ctx context.Context, serial string) ([]Process, error) {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "ps", "-A")
	if errors.Is(err, execx.ErrExit) {
		res, err = a.r.Run(ctx, "adb", "-s", serial, "shell", "ps")
	}
	switch {
	case err == nil:
		return parsePSAll(res.Stdout), nil
	case errors.Is(err, execx.ErrExit):
		return nil, nil
	default:
		return nil, wrapADB("apps", err)
	}
}

func (a *ADB) Pidof(ctx context.Context, serial, pkg string) ([]Process, error) {
	if serial == "" || pkg == "" {
		return nil, fmt.Errorf("device: pidof: %w", ErrUsage)
	}

	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "pidof", pkg)
	var pids []int
	switch {
	case err == nil:
		pids = parsePidof(res.Stdout)
	case errors.Is(err, execx.ErrExit):
		// dead / missing package: pidof exits 1
	default:
		return nil, wrapADB("pidof", err)
	}

	fromPS, err := a.ps(ctx, serial, pkg)
	if err != nil {
		return nil, err
	}
	return mergeProcesses(pkg, pids, fromPS), nil
}

func (a *ADB) ps(ctx context.Context, serial, pkg string) ([]Process, error) {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "ps", "-A")
	if errors.Is(err, execx.ErrExit) {
		res, err = a.r.Run(ctx, "adb", "-s", serial, "shell", "ps")
	}
	switch {
	case err == nil:
		return parsePS(res.Stdout, pkg), nil
	case errors.Is(err, execx.ErrExit):
		return nil, nil
	default:
		return nil, wrapADB("pidof", err)
	}
}

func wrapADB(op string, err error) error {
	if errors.Is(err, execx.ErrNotFound) {
		return fmt.Errorf("device: %s: %w", op, fmt.Errorf("%w: adb", ErrToolMissing))
	}
	return fmt.Errorf("device: %s: %w", op, err)
}
