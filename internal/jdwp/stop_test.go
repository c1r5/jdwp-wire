package jdwp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/logging"
)

func TestForceStop(t *testing.T) {
	t.Parallel()
	var cmds []string
	var log bytes.Buffer
	a := New(&execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		cmds = append(cmds, strings.Join(append([]string{name}, args...), " "))
		return execx.Result{}, nil
	}}, nil)
	a.SetLog(logging.New(&log))
	if err := a.ForceStop("emulator-5554", "com.alvo", 8700); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmds, "\n")
	for _, want := range []string{
		"adb -s emulator-5554 shell am force-stop com.alvo",
		"adb -s emulator-5554 shell am clear-debug-app",
		"adb -s emulator-5554 forward --remove tcp:8700",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}
	text := log.String()
	if !strings.Contains(text, "[ok] [stop] force-stop com.alvo") || !strings.Contains(text, "[ok] [stop] clear-debug-app com.alvo") || !strings.Contains(text, "[ok] [stop] remove forward tcp:8700") {
		t.Fatalf("log %s", text)
	}
}

func TestForceStopUsage(t *testing.T) {
	t.Parallel()
	a := New(&execx.Fake{RunFn: func(context.Context, string, ...string) (execx.Result, error) {
		t.Fatal("adb")
		return execx.Result{}, nil
	}}, nil)
	if err := a.ForceStop("", "com.alvo", 8700); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("err %v", err)
	}
}
