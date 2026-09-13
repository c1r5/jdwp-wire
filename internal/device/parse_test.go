package device

import (
	"testing"
)

const devicesLong = `List of devices attached
emulator-5554          device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64
R58Mxxx                device usb:1-2 product:a71 model:SM_A715F
BAD1                   unauthorized
emulator-5556          offline

`

const devicesShort = `List of devices attached
emulator-5554	device
R58Mxxx	device
BAD1	unauthorized
emulator-5556	offline
`

func TestParseDevicesLong(t *testing.T) {
	t.Parallel()
	got := parseDevices(devicesLong)
	want := []Device{
		{Serial: "emulator-5554", State: StateDevice, Kind: KindEmulator, Model: "sdk_gphone64_x86_64"},
		{Serial: "R58Mxxx", State: StateDevice, Kind: KindUSB, Model: "SM_A715F"},
		{Serial: "BAD1", State: StateUnauthorized, Kind: KindUnknown},
		{Serial: "emulator-5556", State: StateOffline, Kind: KindEmulator},
	}
	assertDevices(t, got, want)
}

func TestParseDevicesShort(t *testing.T) {
	t.Parallel()
	got := parseDevices(devicesShort)
	want := []Device{
		{Serial: "emulator-5554", State: StateDevice, Kind: KindEmulator},
		{Serial: "R58Mxxx", State: StateDevice, Kind: KindUnknown},
		{Serial: "BAD1", State: StateUnauthorized, Kind: KindUnknown},
		{Serial: "emulator-5556", State: StateOffline, Kind: KindEmulator},
	}
	assertDevices(t, got, want)
}

func TestParseDevicesEmpty(t *testing.T) {
	t.Parallel()
	got := parseDevices("List of devices attached\n\n")
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}

func TestParseDevicesUnknownState(t *testing.T) {
	t.Parallel()
	got := parseDevices("List of devices attached\nABC connecting\n")
	want := []Device{{Serial: "ABC", State: StateUnknown, Kind: KindUnknown}}
	assertDevices(t, got, want)
}

func TestParseDevicesNoHeader(t *testing.T) {
	t.Parallel()
	got := parseDevices("emulator-5554 device\n")
	want := []Device{{Serial: "emulator-5554", State: StateDevice, Kind: KindEmulator}}
	assertDevices(t, got, want)
}

func assertDevices(t *testing.T, got, want []Device) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d\ngot  %#v\nwant %#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("i=%d got %#v, want %#v", i, got[i], want[i])
		}
	}
}
