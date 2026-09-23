package attach

import (
	"context"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
)

// Run resolves the device and forwards JDWP for an already-installed package.
func Run(ctx context.Context, dev device.Client, client jdwp.Client, serial, pkg string, port int) (jdwp.Session, error) {
	d, err := dev.Resolve(ctx, serial)
	if err != nil {
		return jdwp.Session{}, err
	}
	return client.Attach(ctx, d.Serial, pkg, port)
}
