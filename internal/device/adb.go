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

func wrapADB(op string, err error) error {
	if errors.Is(err, execx.ErrNotFound) {
		return fmt.Errorf("device: %s: %w", op, fmt.Errorf("%w: adb", ErrToolMissing))
	}
	return fmt.Errorf("device: %s: %w", op, err)
}
