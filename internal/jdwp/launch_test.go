package jdwp

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
)

func withTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSetDebugApp(t *testing.T) {
	t.Parallel()
	var got []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		got = append([]string{name}, args...)
		return execx.Result{}, nil
	}}
	if err := New(r, &device.Fake{}).setDebugApp(withTimeout(t), "emu", "com.alvo"); err != nil {
		t.Fatal(err)
	}
	want := []string{"adb", "-s", "emu", "shell", "am", "set-debug-app", "-w", "com.alvo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestLaunchUsesResolvedActivity(t *testing.T) {
	t.Parallel()
	var cmds [][]string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		cmds = append(cmds, append([]string{name}, args...))
		if len(args) > 0 && strings.Contains(strings.Join(args, " "), "resolve-activity") {
			return execx.Result{Stdout: "com.alvo/.MainActivity\n"}, nil
		}
		return execx.Result{}, nil
	}}
	act, err := New(r, &device.Fake{}).launch(withTimeout(t), "emu", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if act != "com.alvo/.MainActivity" {
		t.Fatalf("activity %q", act)
	}
	if len(cmds) != 2 {
		t.Fatalf("cmds %v", cmds)
	}
	wantStart := []string{"adb", "-s", "emu", "shell", "am", "start", "-D", "-n", "com.alvo/.MainActivity"}
	if !reflect.DeepEqual(cmds[1], wantStart) {
		t.Fatalf("start %v", cmds[1])
	}
}

func TestLaunchFallsBackToMonkey(t *testing.T) {
	t.Parallel()
	var last []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		last = append([]string{name}, args...)
		if strings.Contains(strings.Join(args, " "), "resolve-activity") {
			return execx.Result{Stdout: "No activity found\n"}, nil
		}
		return execx.Result{}, nil
	}}
	act, err := New(r, &device.Fake{}).launch(withTimeout(t), "emu", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if act != "" {
		t.Fatalf("activity %q", act)
	}
	want := []string{"adb", "-s", "emu", "shell", "monkey", "-p", "com.alvo", "-c", "android.intent.category.LAUNCHER", "1"}
	if !reflect.DeepEqual(last, want) {
		t.Fatalf("monkey %v", last)
	}
}

func TestSetDebugAppToolMissing(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{}, execx.ErrNotFound
	}}
	err := New(r, &device.Fake{}).setDebugApp(withTimeout(t), "emu", "com.alvo")
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err=%v", err)
	}
}
