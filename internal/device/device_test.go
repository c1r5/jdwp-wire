package device

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func withTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestDeviceUsable(t *testing.T) {
	t.Parallel()
	if !(Device{State: StateDevice}).Usable() {
		t.Fatal("device state should be usable")
	}
	for _, st := range []State{StateOffline, StateUnauthorized, StateUnknown} {
		if (Device{State: st}).Usable() {
			t.Fatalf("state %q should not be usable", st)
		}
	}
}

func TestFakeResolve(t *testing.T) {
	t.Parallel()
	emu := Device{Serial: "emulator-5554", State: StateDevice, Kind: KindEmulator, Model: "sdk"}
	usb := Device{Serial: "R58Mxxx", State: StateDevice, Kind: KindUSB, Model: "SM_A715F"}
	unauth := Device{Serial: "BAD1", State: StateUnauthorized, Kind: KindUSB}
	offline := Device{Serial: "emulator-5556", State: StateOffline, Kind: KindEmulator}

	tests := []struct {
		name    string
		devs    []Device
		serial  string
		want    Device
		wantErr error
		errText string
	}{
		{
			name:    "zero devices empty serial",
			wantErr: ErrNoDevice,
		},
		{
			name:    "zero devices named serial",
			serial:  "emulator-5554",
			wantErr: ErrNoDevice,
		},
		{
			name:    "only unusable empty serial",
			devs:    []Device{unauth, offline},
			wantErr: ErrNoDevice,
			errText: "unauthorized",
		},
		{
			name:    "only unusable matching serial",
			devs:    []Device{unauth},
			serial:  "BAD1",
			wantErr: ErrDeviceUnusable,
			errText: "BAD1",
		},
		{
			name:    "only unusable unknown serial",
			devs:    []Device{unauth},
			serial:  "nope",
			wantErr: ErrNoDevice,
		},
		{
			name:    "one usable empty serial",
			devs:    []Device{emu, unauth},
			want:    emu,
			wantErr: nil,
		},
		{
			name:   "one usable matching serial",
			devs:   []Device{emu},
			serial: "emulator-5554",
			want:   emu,
		},
		{
			name:    "one usable wrong serial",
			devs:    []Device{emu},
			serial:  "R58Mxxx",
			wantErr: ErrDeviceNotFound,
			errText: "R58Mxxx",
		},
		{
			name:    "n usable empty serial",
			devs:    []Device{emu, usb},
			wantErr: ErrAmbiguousDevice,
			errText: "emulator-5554",
		},
		{
			name:   "n usable matching serial",
			devs:   []Device{emu, usb},
			serial: "R58Mxxx",
			want:   usb,
		},
		{
			name:    "n devices matching unusable serial",
			devs:    []Device{emu, usb, offline},
			serial:  "emulator-5556",
			wantErr: ErrDeviceUnusable,
			errText: "offline",
		},
		{
			name:    "n usable unknown serial",
			devs:    []Device{emu, usb},
			serial:  "missing",
			wantErr: ErrDeviceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &Fake{Devices: tt.devs}
			got, err := f.Resolve(withTimeout(t), tt.serial)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if tt.errText != "" && (err == nil || !strings.Contains(err.Error(), tt.errText)) {
					t.Fatalf("err = %v, want substring %q", err, tt.errText)
				}
				if err == nil || !strings.Contains(err.Error(), "device:") {
					t.Fatalf("err = %v, want device: prefix", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestFakePidof(t *testing.T) {
	t.Parallel()
	procs := []Process{
		{PID: 10, Package: "com.alvo"},
		{PID: 11, Package: "com.alvo:remote"},
	}
	f := &Fake{Procs: map[string][]Process{"com.alvo": procs}}

	got, err := f.Pidof(withTimeout(t), "emulator-5554", "com.alvo")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(got) != 2 || got[0] != procs[0] || got[1] != procs[1] {
		t.Fatalf("got %#v, want %#v", got, procs)
	}

	empty, err := f.Pidof(withTimeout(t), "emulator-5554", "com.other")
	if err != nil {
		t.Fatalf("missing pkg err = %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("missing pkg got %#v, want empty", empty)
	}

	_, err = f.Pidof(withTimeout(t), "emulator-5554", "")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("empty pkg err = %v, want ErrUsage", err)
	}
	_, err = f.Pidof(withTimeout(t), "", "com.alvo")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("empty serial err = %v, want ErrUsage", err)
	}
}

var _ Client = (*Fake)(nil)
