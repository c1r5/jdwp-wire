package device

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestADBListLong(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		if name != "adb" || len(args) != 2 || args[0] != "devices" || args[1] != "-l" {
			t.Fatalf("unexpected run %s %v", name, args)
		}
		return execx.Result{Stdout: devicesLong}, nil
	}}
	got, err := NewADB(r).List(withTimeout(t))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := parseDevices(devicesLong)
	assertDevices(t, got, want)
}

func TestADBListFallbackShort(t *testing.T) {
	t.Parallel()
	var calls []string
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		key := strings.Join(append([]string{name}, args...), " ")
		calls = append(calls, key)
		if key == "adb devices -l" {
			return execx.Result{ExitCode: 1, Stderr: "unknown option -l"}, execx.ErrExit
		}
		if key == "adb devices" {
			return execx.Result{Stdout: devicesShort}, nil
		}
		t.Fatalf("unexpected run %s", key)
		return execx.Result{}, nil
	}}
	got, err := NewADB(r).List(withTimeout(t))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want devices -l then devices", calls)
	}
	assertDevices(t, got, parseDevices(devicesShort))
}

func TestADBListToolMissing(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{}, execx.ErrNotFound
	}}
	_, err := NewADB(r).List(withTimeout(t))
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err = %v, want ErrToolMissing", err)
	}
	if err == nil || !strings.Contains(err.Error(), "device: list:") {
		t.Fatalf("err = %v, want device: list: prefix", err)
	}
}

func TestADBListNoDeadline(t *testing.T) {
	t.Parallel()
	r := execx.Exec{}
	_, err := NewADB(r).List(context.Background())
	if !errors.Is(err, execx.ErrNoDeadline) {
		t.Fatalf("err = %v, want ErrNoDeadline", err)
	}
}

func TestADBResolveUsesList(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{Stdout: devicesLong}, nil
	}}
	got, err := NewADB(r).Resolve(withTimeout(t), "R58Mxxx")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Serial != "R58Mxxx" || got.Kind != KindUSB || got.Model != "SM_A715F" {
		t.Fatalf("got %#v", got)
	}
}

func TestADBResolveAmbiguous(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{Stdout: devicesLong}, nil
	}}
	_, err := NewADB(r).Resolve(withTimeout(t), "")
	if !errors.Is(err, ErrAmbiguousDevice) {
		t.Fatalf("err = %v, want ErrAmbiguousDevice", err)
	}
	if err == nil || !strings.Contains(err.Error(), "emulator-5554") || !strings.Contains(err.Error(), "R58Mxxx") {
		t.Fatalf("err = %v, want usable serials", err)
	}
}
