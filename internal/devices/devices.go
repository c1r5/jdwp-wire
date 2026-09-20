package devices

import (
	"context"

	"github.com/c1r5/jdwp-wire/internal/device"
)

// List returns connected devices via the device client.
func List(ctx context.Context, c device.Client) ([]device.Device, error) {
	return c.List(ctx)
}
