package pull

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestRunLocalRequiresPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Deps{
		Device: &device.Fake{},
		APK:    &apk.Fake{},
		CWD:    dir,
	}, Request{Target: src})
	if !errors.Is(err, apk.ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunPullsPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fake := &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
	}}}
	res, err := Run(context.Background(), Deps{
		Device: fake,
		APK: &apk.Fake{PullFn: func(_ context.Context, serial, pkg string, _ workspace.Layout) (apk.Artifact, error) {
			if serial != "emulator-5554" || pkg != "com.alvo" {
				t.Fatalf("pull %s %s", serial, pkg)
			}
			return apk.Artifact{Package: pkg, APK: filepath.Join(dir, "base.apk")}, nil
		}},
		CWD: dir,
	}, Request{Target: "com.alvo"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Artifact.Package != "com.alvo" || res.DecodeDir != "" {
		t.Fatalf("%+v", res)
	}
}

func TestRunPullsByIndex(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fake := &device.Fake{
		Devices: []device.Device{{Serial: "emulator-5554", State: device.StateDevice}},
		Apps: []device.App{
			{Package: "com.zeta", Label: "Zeta"},
			{Package: "com.alpha", Label: "Alpha"},
		},
	}
	res, err := Run(context.Background(), Deps{
		Device: fake,
		APK: &apk.Fake{PullFn: func(_ context.Context, serial, pkg string, _ workspace.Layout) (apk.Artifact, error) {
			if serial != "emulator-5554" || pkg != "com.alpha" {
				t.Fatalf("pull %s %s", serial, pkg)
			}
			return apk.Artifact{Package: pkg, APK: filepath.Join(dir, "base.apk")}, nil
		}},
		CWD: dir,
	}, Request{Target: "01"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Artifact.Package != "com.alpha" {
		t.Fatalf("%+v", res)
	}
	_, err = Run(context.Background(), Deps{
		Device: fake,
		APK:    &apk.Fake{},
		CWD:    dir,
	}, Request{Target: "9"})
	if !errors.Is(err, device.ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunNumericFileIsLocalAPK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "1")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Deps{
		Device: &device.Fake{Apps: []device.App{{Package: "com.alpha", Label: "Alpha"}}},
		APK:    &apk.Fake{},
		CWD:    dir,
	}, Request{Target: src})
	if !errors.Is(err, apk.ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}
