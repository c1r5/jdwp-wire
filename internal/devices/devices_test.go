package devices

import (
	"context"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestList(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
		Model:  "phone",
	}}}
	got, err := List(context.Background(), fake)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Serial != "emulator-5554" {
		t.Fatalf("%+v", got)
	}
}
