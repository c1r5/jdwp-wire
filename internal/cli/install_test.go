package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestInstallHuman(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var gotSerial, gotAPK string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk: &apk.Fake{
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, serial, apkPath string) error {
				gotSerial, gotAPK = serial, apkPath
				return nil
			},
		},
		args: []string{"install", src},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if gotSerial != "emulator-5554" || gotAPK != src {
		t.Fatalf("install %s %s", gotSerial, gotAPK)
	}
	if !strings.Contains(stdout.String(), "[ok] install:") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestInstallJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk: &apk.Fake{
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _, _ string) error {
				return nil
			},
		},
		args: []string{"install", src, "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		APK    string `json:"apk"`
		Serial string `json:"serial"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.APK != src || got.Serial != "emulator-5554" {
		t.Fatalf("%+v", got)
	}
}

func TestInstallNoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk:    &apk.Fake{},
		args:   []string{"install"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestInstallNoDevice(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: &device.Fake{},
		apk:    &apk.Fake{},
		args:   []string{"install", src},
	})
	if code != ExitNoDevice {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitNoDevice, stderr.String())
	}
}

func TestInstallToolMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk: &apk.Fake{
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _, _ string) error {
				return apk.ErrToolMissing
			},
		},
		args: []string{"install", src},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitToolMissing, stderr.String())
	}
}

func TestInstallMissingFile(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk:    &apk.Fake{},
		args:   []string{"install", filepath.Join(t.TempDir(), "nope.apk")},
	})
	if code == ExitOK {
		t.Fatal("expected failure for missing apk")
	}
}

func TestInstallSignsAPK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var steps []string
	var gotKS string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			BuildFn: func(_ context.Context, _, _ string) (apk.Artifact, error) {
				t.Fatal("encode must not run for an APK file")
				return apk.Artifact{}, nil
			},
			SignFn: func(_ context.Context, apkPath, keystore string) (apk.Artifact, error) {
				steps = append(steps, "sign")
				gotKS = keystore
				if apkPath != src {
					t.Fatalf("sign %s", apkPath)
				}
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, serial, apkPath string) error {
				steps = append(steps, "install")
				if serial != "emulator-5554" || apkPath != src {
					t.Fatalf("install %s %s", serial, apkPath)
				}
				return nil
			},
		},
		args: []string{"install", src},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if strings.Join(steps, ",") != "sign,install" {
		t.Fatalf("steps %v", steps)
	}
	if gotKS != filepath.Join(dir, ".jdt", "debug.keystore") {
		t.Fatalf("keystore %s", gotKS)
	}
	out := stdout.String()
	if strings.Contains(out, "[ok] encode:") {
		t.Fatalf("encode line on apk: %q", out)
	}
	if !strings.Contains(out, "[ok] sign: "+src) {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[ok] install: "+src) {
		t.Fatalf("stdout=%q", out)
	}
}

func TestInstallEncodeFromDecodeDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantAPK := filepath.Join(dir, ".jdt", "com.alvo", "patched", "base.apk")
	var steps []string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			BuildFn: func(_ context.Context, decodedDir, outAPK string) (apk.Artifact, error) {
				steps = append(steps, "encode")
				if decodedDir != decode || outAPK != wantAPK {
					t.Fatalf("build %s %s", decodedDir, outAPK)
				}
				return apk.Artifact{APK: outAPK}, nil
			},
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				steps = append(steps, "sign")
				if apkPath != wantAPK {
					t.Fatalf("sign %s", apkPath)
				}
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _, apkPath string) error {
				steps = append(steps, "install")
				if apkPath != wantAPK {
					t.Fatalf("install %s", apkPath)
				}
				return nil
			},
		},
		args: []string{"install", decode},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if strings.Join(steps, ",") != "encode,sign,install" {
		t.Fatalf("steps %v", steps)
	}
	out := stdout.String()
	if !strings.Contains(out, "[ok] encode: "+wantAPK) {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[ok] sign: "+wantAPK) {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[ok] install: "+wantAPK) {
		t.Fatalf("stdout=%q", out)
	}
}

func TestInstallEncodeFromPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "AndroidManifest.xml"), []byte(`<manifest package="com.alvo"/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	wantAPK := filepath.Join(dir, ".jdt", "com.alvo", "patched", "base.apk")
	var builtFrom string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			BuildFn: func(_ context.Context, decodedDir, outAPK string) (apk.Artifact, error) {
				builtFrom = decodedDir
				if outAPK != wantAPK {
					t.Fatalf("out %s", outAPK)
				}
				return apk.Artifact{APK: outAPK}, nil
			},
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _, _ string) error { return nil },
		},
		args: []string{"install", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if builtFrom != decode {
		t.Fatalf("built from %q", builtFrom)
	}
}

func TestInstallDecodeDirWithPackageFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, "smali")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantAPK := filepath.Join(dir, ".jdt", "com.alvo", "patched", "base.apk")
	var builtFrom, builtTo string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			BuildFn: func(_ context.Context, decodedDir, outAPK string) (apk.Artifact, error) {
				builtFrom, builtTo = decodedDir, outAPK
				return apk.Artifact{APK: outAPK}, nil
			},
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _, _ string) error { return nil },
		},
		args: []string{"install", decode, "--package", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if builtFrom != decode || builtTo != wantAPK {
		t.Fatalf("build %s %s", builtFrom, builtTo)
	}
}

func TestInstallDecodeDirNeedsPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, "smali")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			BuildFn: func(_ context.Context, _, _ string) (apk.Artifact, error) {
				t.Fatal("build without --package")
				return apk.Artifact{}, nil
			},
			SignFn: func(_ context.Context, apkPath, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: apkPath}, nil
			},
			InstallFn: func(_ context.Context, _, _ string) error { return nil },
		},
		args: []string{"install", decode},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestInstallEncodeToolMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decode := filepath.Join(dir, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			BuildFn: func(_ context.Context, _, _ string) (apk.Artifact, error) {
				return apk.Artifact{}, apk.ErrToolMissing
			},
		},
		args: []string{"install", decode},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitToolMissing, stderr.String())
	}
}
