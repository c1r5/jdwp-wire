package execx

import (
	"context"
	"errors"
	"testing"
)

func TestFake_NilRunFn(t *testing.T) {
	t.Parallel()
	var f Fake
	_, err := f.Run(context.Background(), "adb", "devices")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestFake_RunFn(t *testing.T) {
	t.Parallel()
	want := Result{Stdout: "ok", ExitCode: 0}
	var sawName string
	var sawArgs []string
	f := Fake{
		RunFn: func(ctx context.Context, name string, args ...string) (Result, error) {
			sawName = name
			sawArgs = append([]string(nil), args...)
			return want, nil
		},
	}
	got, err := f.Run(context.Background(), "adb", "devices", "-l")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if sawName != "adb" {
		t.Fatalf("name = %q", sawName)
	}
	if len(sawArgs) != 2 || sawArgs[0] != "devices" || sawArgs[1] != "-l" {
		t.Fatalf("args = %#v", sawArgs)
	}
}

var _ Runner = (*Fake)(nil)
var _ Runner = Exec{}
