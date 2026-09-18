package jdwp

import (
	"context"
	"errors"
	"testing"
)

func TestFakeAttach(t *testing.T) {
	t.Parallel()
	want := Session{
		Package:  "com.alvo",
		Serial:   "emulator-5554",
		PID:      1234,
		Port:     8700,
		Activity: "com.alvo/.MainActivity",
	}
	f := &Fake{
		AttachFn: func(_ context.Context, serial, pkg string, port int) (Session, error) {
			if serial != "emulator-5554" || pkg != "com.alvo" || port != 8700 {
				t.Fatalf("args %s %s %d", serial, pkg, port)
			}
			return want, nil
		},
	}
	got, err := f.Attach(context.Background(), "emulator-5554", "com.alvo", 8700)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestFakeReset(t *testing.T) {
	t.Parallel()
	var gotSerial, gotPkg string
	var gotPort int
	f := &Fake{
		ResetFn: func(_ context.Context, serial, pkg string, port int) error {
			gotSerial, gotPkg, gotPort = serial, pkg, port
			return nil
		},
	}
	if err := f.Reset(context.Background(), "emulator-5554", "com.alvo", 8700); err != nil {
		t.Fatalf("err=%v", err)
	}
	if gotSerial != "emulator-5554" || gotPkg != "com.alvo" || gotPort != 8700 {
		t.Fatalf("reset %s %s %d", gotSerial, gotPkg, gotPort)
	}
}

func TestFakeNilFn(t *testing.T) {
	t.Parallel()
	var f Fake
	_, err := f.Attach(context.Background(), "s", "p", 8700)
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("attach err=%v", err)
	}
	err = f.Reset(context.Background(), "s", "p", 8700)
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("reset err=%v", err)
	}
}
