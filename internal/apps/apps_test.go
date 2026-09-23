package apps

import (
	"context"
	"errors"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestListOrdersByName(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{
		Devices: []device.Device{{Serial: "emulator-5554", State: device.StateDevice}},
		Apps: []device.App{
			{Package: "com.zeta", Label: "Zeta", PID: 10},
			{Package: "com.b", Label: "Same"},
			{Package: "com.a", Label: "Same", PID: -1},
			{Package: "com.plain", PID: 4},
		},
	}
	got, err := List(context.Background(), fake, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"com.plain", "com.a", "com.b", "com.zeta"}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i, id := range want {
		if got[i].Index != i+1 || got[i].Identifier != id {
			t.Fatalf("row %d %+v", i, got[i])
		}
	}
	if got[0].Name != "com.plain" || got[0].PID != 4 {
		t.Fatalf("plain %+v", got[0])
	}
	if got[1].PID != 0 || got[1].Name != "Same" {
		t.Fatalf("same a %+v", got[1])
	}
	pkg, err := PackageAt(got, 2)
	if err != nil || pkg != "com.a" {
		t.Fatalf("at 2 %s %v", pkg, err)
	}
	if _, err := PackageAt(got, 0); !errors.Is(err, device.ErrUsage) {
		t.Fatalf("zero %v", err)
	}
	if _, err := PackageAt(got, 9); !errors.Is(err, device.ErrUsage) {
		t.Fatalf("high %v", err)
	}
}

func TestListNoDevice(t *testing.T) {
	t.Parallel()
	_, err := List(context.Background(), &device.Fake{}, "")
	if !errors.Is(err, device.ErrNoDevice) {
		t.Fatalf("%v", err)
	}
}
