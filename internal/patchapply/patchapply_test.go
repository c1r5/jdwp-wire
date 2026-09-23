package patchapply

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestRunRequiresDirOrAPK(t *testing.T) {
	t.Parallel()
	_, err := Run(context.Background(), Deps{Patch: &patch.Fake{}, APK: &apk.Fake{}}, Request{})
	if !errors.Is(err, patch.ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunAPKDecodesThenApplies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(apkPath, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var applied string
	res, err := Run(context.Background(), Deps{
		CWD: dir,
		APK: &apk.Fake{DecodeFn: func(_ context.Context, src string, layout workspace.Layout) (apk.Decoded, error) {
			if src != apkPath {
				t.Fatalf("src %s", src)
			}
			return apk.Decoded{Package: "com.alvo", Dir: layout.Decode}, nil
		}},
		Patch: &patch.Fake{ApplyFn: func(d string) (patch.Result, error) {
			applied = d
			return patch.Result{Package: "com.alvo", Dir: d, Debuggable: patch.ActionApplied, NSC: patch.ActionApplied}, nil
		}},
	}, Request{APK: apkPath, Package: "com.alvo"})
	if err != nil {
		t.Fatal(err)
	}
	if applied == "" || res.Dir != applied {
		t.Fatalf("applied=%q res=%+v", applied, res)
	}
}
