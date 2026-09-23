package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSmali = "" +
	".class public Lcom/example/Api;\n" +
	".super Ljava/lang/Object;\n" +
	".method public load()V\n" +
	"    .locals 1\n" +
	"    invoke-virtual {v0}, Lokhttp3/Request$Builder;->build()Lokhttp3/Request;\n" +
	"    return-void\n" +
	".end method\n"

func TestTargetsHumanPackage(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	decode := filepath.Join(cwd, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "Api.smali"), []byte(sampleSmali), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    cwd,
		args:   []string{"targets", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if stdout.String() != "com.example.Api#load\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestTargetsJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Api.smali"), []byte(sampleSmali), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    t.TempDir(),
		args:   []string{"targets", dir, "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Dir     string `json:"dir"`
		Targets []struct {
			Class  string   `json:"class"`
			Method string   `json:"method"`
			Kinds  []string `json:"kinds"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != abs || len(got.Targets) != 1 || got.Targets[0].Class != "com.example.Api" || got.Targets[0].Method != "load" {
		t.Fatalf("%+v", got)
	}
	if len(got.Targets[0].Kinds) != 1 || got.Targets[0].Kinds[0] != "okhttp" {
		t.Fatalf("kinds=%v", got.Targets[0].Kinds)
	}
}

func TestTargetsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		args:   []string{"targets", dir},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if stdout.String() != "[skip] targets: no http sinks\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestTargetsUsage(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		args:   []string{"targets"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestTargetsHTTPOff(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		args:   []string{"targets", dir, "--http=false"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestTargetsMissingDecode(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    t.TempDir(),
		args:   []string{"targets", "com.missing"},
	})
	if code == ExitOK || code == ExitUsage {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "targets: decode:") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}
