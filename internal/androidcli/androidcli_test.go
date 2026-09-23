package androidcli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestAvailableMissing(t *testing.T) {
	t.Parallel()
	c := &CLI{
		r: &execx.Fake{},
		look: func(string) (string, error) {
			return "", execx.ErrNotFound
		},
	}
	ok, err := c.Available(context.Background())
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestAvailableLegacy(t *testing.T) {
	t.Parallel()
	c := &CLI{
		r: &execx.Fake{RunFn: func(context.Context, string, ...string) (execx.Result, error) {
			return execx.Result{Stdout: "The android command is deprecated. Use sdkmanager."}, nil
		}},
		look: func(string) (string, error) { return "/sdk/tools/android", nil },
	}
	ok, err := c.Available(context.Background())
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestAvailableCLI(t *testing.T) {
	t.Parallel()
	var got []string
	c := &CLI{
		r: &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
			got = append([]string{name}, args...)
			return execx.Result{Stderr: "Usage: android run [--debug] --apks=PARAM\n"}, &execx.ExitError{ExitCode: 2}
		}},
		look: func(string) (string, error) { return "/usr/bin/android", nil },
	}
	ok, err := c.Available(context.Background())
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if strings.Join(got, " ") != "android run -h" {
		t.Fatalf("cmd %v", got)
	}
}

func TestRunDebugArgs(t *testing.T) {
	t.Parallel()
	var got []string
	c := &CLI{
		r: &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
			got = append([]string{name}, args...)
			return execx.Result{}, nil
		}},
		look: func(string) (string, error) { return "/usr/bin/android", nil },
	}
	err := c.RunDebug(context.Background(), "emulator-5554", []string{"/tmp/base.apk", "/tmp/split.apk"})
	if err != nil {
		t.Fatal(err)
	}
	want := "android run --debug --device=emulator-5554 --apks=/tmp/base.apk,/tmp/split.apk --install-options=-r,-d"
	if strings.Join(got, " ") != want {
		t.Fatalf("cmd %q", strings.Join(got, " "))
	}
}

func TestRunDebugUsage(t *testing.T) {
	t.Parallel()
	c := &CLI{r: &execx.Fake{}, look: func(string) (string, error) { return "android", nil }}
	if err := c.RunDebug(context.Background(), "", []string{"a.apk"}); !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
	if err := c.RunDebug(context.Background(), "emu", nil); !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunDebugExit(t *testing.T) {
	t.Parallel()
	c := &CLI{
		r: &execx.Fake{RunFn: func(context.Context, string, ...string) (execx.Result, error) {
			return execx.Result{Stderr: "INSTALL_FAILED"}, &execx.ExitError{ExitCode: 1, Stderr: "INSTALL_FAILED"}
		}},
		look: func(string) (string, error) { return "android", nil },
	}
	err := c.RunDebug(context.Background(), "emu", []string{"a.apk"})
	if err == nil || errors.Is(err, ErrToolMissing) || !strings.Contains(err.Error(), "INSTALL_FAILED") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunDebugMissing(t *testing.T) {
	t.Parallel()
	c := &CLI{
		r:    &execx.Fake{},
		look: func(string) (string, error) { return "", execx.ErrNotFound },
	}
	err := c.RunDebug(context.Background(), "emu", []string{"a.apk"})
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err=%v", err)
	}
}
