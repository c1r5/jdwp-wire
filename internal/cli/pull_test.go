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
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func testDevice() *device.Fake {
	return &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
		Model:  "phone",
	}}}
}

func TestPullHuman(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk: &apk.Fake{PullFn: func(_ context.Context, serial, pkg string, _ workspace.Layout) (apk.Artifact, error) {
			if serial != "emulator-5554" || pkg != "com.alvo" {
				t.Fatalf("pull %s %s", serial, pkg)
			}
			return apk.Artifact{
				Package:       "com.alvo",
				APK:           ".jdt/com.alvo/apk/base.apk",
				SkippedSplits: []string{"split_config.xxhdpi.apk"},
			}, nil
		}},
		args: []string{"pull", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "[ok] pull: com.alvo") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "base.apk") {
		t.Fatalf("path missing: %q", out)
	}
	if !strings.Contains(out, "[skip] split: split_config.xxhdpi.apk") {
		t.Fatalf("skip missing: %q", out)
	}
}

func TestPullJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk: &apk.Fake{PullFn: func(_ context.Context, _, _ string, _ workspace.Layout) (apk.Artifact, error) {
			return apk.Artifact{Package: "com.alvo", APK: ".jdt/com.alvo/apk/base.apk"}, nil
		}},
		args: []string{"pull", "com.alvo", "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Package       string   `json:"package"`
		APK           string   `json:"apk"`
		SkippedSplits []string `json:"skipped_splits"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Package != "com.alvo" || got.APK != ".jdt/com.alvo/apk/base.apk" {
		t.Fatalf("%+v", got)
	}
	if got.SkippedSplits == nil {
		t.Fatal("skipped_splits null")
	}
}

func TestPullLocalRequiresPackage(t *testing.T) {
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
		apk:    &apk.Fake{},
		cwd:    dir,
		args:   []string{"pull", src},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestPullNoDevice(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: &device.Fake{},
		apk:    &apk.Fake{},
		args:   []string{"pull", "com.alvo"},
	})
	if code != ExitNoDevice {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitNoDevice, stderr.String())
	}
}

func TestPullToolMissing(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk: &apk.Fake{PullFn: func(_ context.Context, _, _ string, _ workspace.Layout) (apk.Artifact, error) {
			return apk.Artifact{}, apk.ErrToolMissing
		}},
		args: []string{"pull", "com.alvo"},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitToolMissing, stderr.String())
	}
}

func TestPullNoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		apk:    &apk.Fake{},
		args:   []string{"pull"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestPullDecode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var decodedFrom, decodeDir string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			PullFn: func(_ context.Context, _, _ string, layout workspace.Layout) (apk.Artifact, error) {
				return apk.Artifact{Package: "com.alvo", APK: filepath.Join(layout.APK, "base.apk")}, nil
			},
			DecodeFn: func(_ context.Context, apkPath string, layout workspace.Layout) (apk.Decoded, error) {
				decodedFrom = apkPath
				decodeDir = layout.Decode
				return apk.Decoded{Package: "com.alvo", Dir: layout.Decode}, nil
			},
		},
		args: []string{"pull", "--decode", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	wantAPK := filepath.Join(dir, ".jdt", "com.alvo", "apk", "base.apk")
	if decodedFrom != wantAPK {
		t.Fatalf("decode src %q want %q", decodedFrom, wantAPK)
	}
	out := stdout.String()
	if !strings.Contains(out, "[ok] pull: com.alvo") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[ok] decode: "+decodeDir) {
		t.Fatalf("stdout=%q", out)
	}
}

func TestPullDecodeJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			PullFn: func(_ context.Context, _, _ string, layout workspace.Layout) (apk.Artifact, error) {
				return apk.Artifact{Package: "com.alvo", APK: filepath.Join(layout.APK, "base.apk")}, nil
			},
			DecodeFn: func(_ context.Context, _ string, layout workspace.Layout) (apk.Decoded, error) {
				return apk.Decoded{Package: "com.alvo", Dir: layout.Decode}, nil
			},
		},
		args: []string{"pull", "--decode", "com.alvo", "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Package       string   `json:"package"`
		APK           string   `json:"apk"`
		SkippedSplits []string `json:"skipped_splits"`
		Decode        string   `json:"decode"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	wantDecode := filepath.Join(dir, ".jdt", "com.alvo", "decode")
	if got.Decode != wantDecode {
		t.Fatalf("%+v", got)
	}
}

func TestPullDecodeLocalAPK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var decodedFrom string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			CopyFn: func(_ context.Context, copySrc, pkg string, layout workspace.Layout) (apk.Artifact, error) {
				if copySrc != src || pkg != "com.alvo" {
					t.Fatalf("copy %s %s", copySrc, pkg)
				}
				return apk.Artifact{Package: pkg, APK: filepath.Join(layout.APK, "base.apk")}, nil
			},
			DecodeFn: func(_ context.Context, apkPath string, layout workspace.Layout) (apk.Decoded, error) {
				decodedFrom = apkPath
				return apk.Decoded{Package: "com.alvo", Dir: layout.Decode}, nil
			},
		},
		args: []string{"pull", "--decode", src, "--package", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if decodedFrom == "" {
		t.Fatal("decode not called")
	}
	if !strings.Contains(stdout.String(), "[ok] decode:") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestPullDecodeToolMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		device: testDevice(),
		apk: &apk.Fake{
			PullFn: func(_ context.Context, _, _ string, layout workspace.Layout) (apk.Artifact, error) {
				return apk.Artifact{Package: "com.alvo", APK: filepath.Join(layout.APK, "base.apk")}, nil
			},
			DecodeFn: func(_ context.Context, _ string, _ workspace.Layout) (apk.Decoded, error) {
				return apk.Decoded{}, apk.ErrToolMissing
			},
		},
		args: []string{"pull", "--decode", "com.alvo"},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitToolMissing, stderr.String())
	}
}
