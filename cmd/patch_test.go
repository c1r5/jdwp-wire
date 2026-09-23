package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestPatchHuman(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		patch: &patch.Fake{ApplyFn: func(d string) (patch.Result, error) {
			if d != dir {
				t.Fatalf("dir %s", d)
			}
			return patch.Result{
				Package:    "com.alvo",
				Dir:        dir,
				Debuggable: patch.ActionApplied,
				NSC:        patch.ActionSkipped,
			}, nil
		}},
		args: []string{"patch", dir},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "[ok] patch: debuggable") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[skip] patch: nsc already trusts user CA") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestPatchJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		patch: &patch.Fake{ApplyFn: func(string) (patch.Result, error) {
			return patch.Result{
				Package:    "com.alvo",
				Dir:        dir,
				Debuggable: patch.ActionApplied,
				NSC:        patch.ActionSkipped,
			}, nil
		}},
		args: []string{"patch", dir, "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Package    string `json:"package"`
		Dir        string `json:"dir"`
		Debuggable string `json:"debuggable"`
		NSC        string `json:"nsc"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Package != "com.alvo" || got.Debuggable != "applied" || got.NSC != "skipped" {
		t.Fatalf("%+v", got)
	}
}

func TestPatchAPKRequiresPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(apkPath, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		patch:  &patch.Fake{},
		apk:    &apk.Fake{},
		args:   []string{"patch", "--apk", apkPath},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestPatchAPKDecodesThenApplies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(apkPath, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var decoded, applied string
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		apk: &apk.Fake{DecodeFn: func(_ context.Context, src string, layout workspace.Layout) (apk.Decoded, error) {
			decoded = layout.Decode
			if src != apkPath {
				t.Fatalf("src %s", src)
			}
			return apk.Decoded{Package: "com.alvo", Dir: layout.Decode}, nil
		}},
		patch: &patch.Fake{ApplyFn: func(d string) (patch.Result, error) {
			applied = d
			return patch.Result{
				Package:    "com.alvo",
				Dir:        d,
				Debuggable: patch.ActionApplied,
				NSC:        patch.ActionApplied,
			}, nil
		}},
		args: []string{"patch", "--apk", apkPath, "--package", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if decoded == "" || applied != decoded {
		t.Fatalf("decoded=%q applied=%q", decoded, applied)
	}
}

func TestPatchNoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		patch:  &patch.Fake{},
		args:   []string{"patch"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestPatchToolMissingOnAPK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(apkPath, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    dir,
		apk: &apk.Fake{DecodeFn: func(_ context.Context, _ string, _ workspace.Layout) (apk.Decoded, error) {
			return apk.Decoded{}, apk.ErrToolMissing
		}},
		patch: &patch.Fake{},
		args:  []string{"patch", "--apk", apkPath, "--package", "com.alvo"},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitToolMissing, stderr.String())
	}
}
