package reset

import (
	"context"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
)

func TestRun(t *testing.T) {
	t.Parallel()
	dev := &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
	}}}
	res, err := Run(context.Background(), dev, &jdwp.Fake{ResetFn: func(_ context.Context, serial, pkg string, port int) error {
		if serial != "emulator-5554" || pkg != "com.alvo" || port != 8700 {
			t.Fatalf("reset %s %s %d", serial, pkg, port)
		}
		return nil
	}}, "", "com.alvo", 8700)
	if err != nil {
		t.Fatal(err)
	}
	if res.Package != "com.alvo" || res.Serial != "emulator-5554" || res.Port != 8700 {
		t.Fatalf("%+v", res)
	}
}
