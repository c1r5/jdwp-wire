package apk

import (
	"context"
	"errors"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestFakePull(t *testing.T) {
	t.Parallel()
	want := Artifact{Package: "com.alvo", APK: "/tmp/base.apk"}
	f := &Fake{
		PullFn: func(_ context.Context, serial, pkg string, _ workspace.Layout) (Artifact, error) {
			if serial != "emulator-5554" || pkg != "com.alvo" {
				t.Fatalf("args %s %s", serial, pkg)
			}
			return want, nil
		},
	}
	got, err := f.Pull(context.Background(), "emulator-5554", "com.alvo", workspace.Layout{})
	if err != nil || got.APK != want.APK {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestFakeNilFn(t *testing.T) {
	t.Parallel()
	var f Fake
	_, err := f.Pull(context.Background(), "s", "p", workspace.Layout{})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}
