package apps

import (
	"context"
	"errors"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestListOrdersRunningThenName(t *testing.T) {
	t.Parallel()
	fake := listFake()
	got, err := List(context.Background(), fake, "", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"com.plain", "com.zeta", "com.a", "com.b"}
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
	if got[1].PID != 10 || got[1].Identifier != "com.zeta" {
		t.Fatalf("zeta %+v", got[1])
	}
	if got[2].PID != 0 || got[2].Name != "Same" || got[2].Identifier != "com.a" {
		t.Fatalf("same a %+v", got[2])
	}
	pkg, err := PackageAt(got, 3)
	if err != nil || pkg != "com.a" {
		t.Fatalf("at 3 %s %v", pkg, err)
	}
	if _, err := PackageAt(got, 0); !errors.Is(err, device.ErrUsage) {
		t.Fatalf("zero %v", err)
	}
	if _, err := PackageAt(got, 9); !errors.Is(err, device.ErrUsage) {
		t.Fatalf("high %v", err)
	}
}

func TestListSkipsSystemUnlessAsked(t *testing.T) {
	t.Parallel()
	fake := listFake()
	user, err := List(context.Background(), fake, "", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range user {
		if e.Identifier == "com.android.settings" {
			t.Fatalf("system app in default list %+v", user)
		}
	}
	all, err := List(context.Background(), fake, "", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"com.plain", "com.android.settings", "com.zeta", "com.a", "com.b"}
	if len(all) != len(want) {
		t.Fatalf("%+v", all)
	}
	for i, id := range want {
		if all[i].Index != i+1 || all[i].Identifier != id {
			t.Fatalf("row %d %+v", i, all[i])
		}
	}
	if all[1].PID != 50 {
		t.Fatalf("settings %+v", all[1])
	}
}

func listFake() *device.Fake {
	return &device.Fake{
		Devices: []device.Device{{Serial: "emulator-5554", State: device.StateDevice}},
		Apps: []device.App{
			{Package: "com.zeta", Label: "Zeta", PID: 10},
			{Package: "com.b", Label: "Same"},
			{Package: "com.a", Label: "Same", PID: -1},
			{Package: "com.plain", PID: 4},
			{Package: "com.android.settings", Label: "Settings", PID: 50, System: true},
		},
	}
}

func TestListNoDevice(t *testing.T) {
	t.Parallel()
	_, err := List(context.Background(), &device.Fake{}, "", false)
	if !errors.Is(err, device.ErrNoDevice) {
		t.Fatalf("%v", err)
	}
}
