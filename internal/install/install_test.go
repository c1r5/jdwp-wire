package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
)

func testDevice() *device.Fake {
	return &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
	}}}
}

func TestRunSignsAPK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var gotSerial, gotAPK string
	res, err := Run(context.Background(), Deps{
		Device: testDevice(),
		APK: &apk.Fake{
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, serial string, apks ...string) error {
				gotSerial, gotAPK = serial, apks[0]
				return nil
			},
		},
		CWD: dir,
	}, Request{Target: src})
	if err != nil {
		t.Fatal(err)
	}
	if gotSerial != "emulator-5554" || gotAPK != src || res.Encoded {
		t.Fatalf("serial=%s apk=%s res=%+v", gotSerial, gotAPK, res)
	}
}

func TestRunDecodeDirNeedsPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, "smali")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Deps{
		Device: testDevice(),
		APK:    &apk.Fake{},
		CWD:    dir,
	}, Request{Target: decode})
	if !errors.Is(err, apk.ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunSplitsFromPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, ".jdt", "com.alvo", "decode")
	apkDir := filepath.Join(dir, ".jdt", "com.alvo", "apk")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(apkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apkDir, "split_config.xxhdpi.apk"), []byte("split"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantBase := filepath.Join(dir, ".jdt", "com.alvo", "patched", "base.apk")
	wantSplit := filepath.Join(dir, ".jdt", "com.alvo", "patched", "split_config.xxhdpi.apk")
	var installed []string
	res, err := Run(context.Background(), Deps{
		Device: testDevice(),
		APK: &apk.Fake{
			BuildFn: func(_ context.Context, decodedDir, outAPK string) (apk.Artifact, error) {
				if decodedDir != decode || outAPK != wantBase {
					t.Fatalf("build %s %s", decodedDir, outAPK)
				}
				return apk.Artifact{APK: outAPK}, nil
			},
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _ string, apks ...string) error {
				installed = append([]string{}, apks...)
				return nil
			},
		},
		CWD: dir,
	}, Request{Target: "com.alvo"})
	if err != nil {
		t.Fatal(err)
	}
	if res.APK != wantBase || len(res.Splits) != 1 || res.Splits[0] != wantSplit {
		t.Fatalf("%+v", res)
	}
	if len(installed) != 2 || installed[0] != wantBase || installed[1] != wantSplit {
		t.Fatalf("installed %v", installed)
	}
}
