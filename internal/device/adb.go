package device

import (
	"context"
	"errors"
	"fmt"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

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
