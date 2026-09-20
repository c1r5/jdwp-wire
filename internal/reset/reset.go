package reset

import (
	"context"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
)

// Result is a completed reset of debug-app + JDWP forward.
type Result struct {
	Package string
	Serial  string
	Port    int
}

// Run clears set-debug-app and removes the JDWP forward.
func Run(ctx context.Context, dev device.Client, client jdwp.Client, serial, pkg string, port int) (Result, error) {
	d, err := dev.Resolve(ctx, serial)
	if err != nil {
		return Result{}, err
	}
	if err := client.Reset(ctx, d.Serial, pkg, port); err != nil {
		return Result{}, err
	}
	return Result{Package: pkg, Serial: d.Serial, Port: port}, nil
}
