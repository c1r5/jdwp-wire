package apk

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestInstallArgs(t *testing.T) {
	t.Parallel()
	var got []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		got = append([]string{name}, args...)
		return execx.Result{}, nil
	}}
	if err := New(r).Install(context.Background(), "emu", "/tmp/a.apk"); err != nil {
		t.Fatal(err)
	}
	want := []string{"adb", "-s", "emu", "install", "-r", "-d", "/tmp/a.apk"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestInstallUsage(t *testing.T) {
	t.Parallel()
	err := New(&execx.Fake{}).Install(context.Background(), "", "/tmp/a.apk")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestInstallToolMissing(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{}, execx.ErrNotFound
	}}
	err := New(r).Install(context.Background(), "emu", "/tmp/a.apk")
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err=%v", err)
	}
}
