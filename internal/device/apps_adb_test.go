package device

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestADBListApps(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		if name != "adb" {
			t.Fatalf("name %s", name)
		}
		key := strings.Join(args, " ")
		switch {
		case strings.Contains(key, "pm list packages -3"):
			return execx.Result{Stdout: "package:com.zeta\npackage:com.alpha\n"}, nil
		case strings.Contains(key, "ps -A"):
			return execx.Result{Stdout: "USER PID PPID NAME\nroot 111 1 com.zeta\nroot 222 1 com.zeta:push\n"}, nil
		case strings.Contains(key, "dumpsys package"):
			return execx.Result{Stdout: "Package [com.alpha] (1):\n  Application Label: Alpha\n"}, nil
		default:
			t.Fatalf("unexpected %s", key)
			return execx.Result{}, nil
		}
	}}
	got, err := NewADB(r).ListApps(withTimeout(t), "emulator-5554", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].Package != "com.zeta" || got[0].PID != 111 || got[0].Label != "" {
		t.Fatalf("zeta %+v", got[0])
	}
	if got[1].Package != "com.alpha" || got[1].PID != 0 || got[1].Label != "Alpha" {
		t.Fatalf("alpha %+v", got[1])
	}
}

func TestADBListAppsDumpsysFailureKeepsPackages(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, _ string, args ...string) (execx.Result, error) {
		key := strings.Join(args, " ")
		switch {
		case strings.Contains(key, "pm list packages -3"):
			return execx.Result{Stdout: "package:com.a\n"}, nil
		case strings.Contains(key, "ps"):
			return execx.Result{ExitCode: 1}, execx.ErrExit
		case strings.Contains(key, "dumpsys"):
			return execx.Result{}, context.DeadlineExceeded
		default:
			t.Fatalf("unexpected %s", key)
			return execx.Result{}, nil
		}
	}}
	got, err := NewADB(r).ListApps(withTimeout(t), "emulator-5554", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Package != "com.a" || got[0].PID != 0 || got[0].Label != "" {
		t.Fatalf("%+v", got)
	}
}

func TestADBListAppsIncludesSystem(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, _ string, args ...string) (execx.Result, error) {
		key := strings.Join(args, " ")
		switch {
		case strings.Contains(key, "pm list packages"):
			if strings.Contains(key, "-3") {
				t.Fatalf("system list filtered to third-party: %s", key)
			}
			return execx.Result{Stdout: "package:com.user\npackage:android\n"}, nil
		case strings.Contains(key, "ps"):
			return execx.Result{Stdout: "USER PID PPID NAME\n"}, nil
		case strings.Contains(key, "dumpsys"):
			return execx.Result{Stdout: ""}, nil
		default:
			t.Fatalf("unexpected %s", key)
			return execx.Result{}, nil
		}
	}}
	got, err := NewADB(r).ListApps(withTimeout(t), "emulator-5554", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Package != "com.user" || got[1].Package != "android" {
		t.Fatalf("%+v", got)
	}
}

func TestADBListAppsUsageAndTool(t *testing.T) {
	t.Parallel()
	adb := NewADB(&execx.Fake{RunFn: func(context.Context, string, ...string) (execx.Result, error) {
		return execx.Result{}, execx.ErrNotFound
	}})
	if _, err := adb.ListApps(withTimeout(t), "", false); !errors.Is(err, ErrUsage) {
		t.Fatalf("empty serial %v", err)
	}
	_, err := adb.ListApps(withTimeout(t), "serial", false)
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err %v", err)
	}
}
