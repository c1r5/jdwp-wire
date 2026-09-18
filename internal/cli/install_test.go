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
		apk: &apk.Fake{InstallFn: func(_ context.Context, serial, apkPath string) error {
			gotSerial, gotAPK = serial, apkPath
			return nil
		}},
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
		apk: &apk.Fake{InstallFn: func(_ context.Context, _, _ string) error {
			return nil
		}},
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
		apk: &apk.Fake{InstallFn: func(_ context.Context, _, _ string) error {
			return apk.ErrToolMissing
		}},
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
