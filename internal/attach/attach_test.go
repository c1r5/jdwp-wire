package attach

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
	sess, err := Run(context.Background(), dev, &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
		return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port}, nil
	}}, "", "com.alvo", 8700)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Serial != "emulator-5554" || sess.Package != "com.alvo" || sess.Port != 8700 {
		t.Fatalf("%+v", sess)
	}
}
