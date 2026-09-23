package androidcli

import (
	"context"
	"fmt"
)

// Fake is an in-memory Client. AvailableFn nil means the CLI is absent.
type Fake struct {
	AvailableFn func(ctx context.Context) (bool, error)
	RunDebugFn  func(ctx context.Context, serial string, apks []string) error
}

func (f *Fake) Available(ctx context.Context) (bool, error) {
	if f == nil || f.AvailableFn == nil {
		return false, nil
	}
	return f.AvailableFn(ctx)
}

func (f *Fake) RunDebug(ctx context.Context, serial string, apks []string) error {
	if f == nil || f.RunDebugFn == nil {
		return fmt.Errorf("androidcli: run: %w", ErrUsage)
	}
	return f.RunDebugFn(ctx, serial, apks)
}

var (
	_ Client = (*CLI)(nil)
	_ Client = (*Fake)(nil)
)
