package execx

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLook_NotFound(t *testing.T) {
	t.Parallel()
	_, err := Look("definitely-not-a-binary-execx-xyz")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("err = %v, want wrapping exec.ErrNotFound", err)
	}
	if err == nil || !strings.Contains(err.Error(), "execx:") {
		t.Fatalf("err = %v, want execx: prefix", err)
	}
}

func TestLook_Found(t *testing.T) {
	t.Parallel()
	path, err := Look("true")
	if err != nil {
		t.Fatalf("Look(true): %v", err)
	}
	if path == "" || !strings.Contains(path, "true") {
		t.Fatalf("path = %q, want a path containing true", path)
	}
}

func withTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestRun_NoDeadline(t *testing.T) {
	t.Parallel()
	_, err := Exec{}.Run(context.Background(), "true")
	if !errors.Is(err, ErrNoDeadline) {
		t.Fatalf("err = %v, want ErrNoDeadline", err)
	}
}

func TestRun_ExitZero(t *testing.T) {
	t.Parallel()
	res, err := Exec{}.Run(withTimeout(t), "true")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRun_Stdout(t *testing.T) {
	t.Parallel()
	res, err := Exec{}.Run(withTimeout(t), "/bin/echo", "hello")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if strings.TrimSpace(res.Stdout) != "hello" {
		t.Fatalf("stdout = %q, want hello", res.Stdout)
	}
	if res.Stderr != "" {
		t.Fatalf("stderr = %q, want empty", res.Stderr)
	}
}

func TestRun_OrphanPipeKeepsSuccessfulOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	script := filepath.Join(dir, "orphan.sh")
	body := "#!/bin/sh\necho 'Usage: android run --debug --apks=PARAM'\nsleep 5 &\nexit 0\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Exec{}.Run(withTimeout(t), script)
	if err != nil {
		t.Fatalf("err = %v\nstdout=%q stderr=%q", err, res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "--apks") || !strings.Contains(res.Stdout, "--debug") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d", res.ExitCode)
	}
}

func TestRun_ExitNonZero(t *testing.T) {
	t.Parallel()
	res, err := Exec{}.Run(withTimeout(t), "false")
	if !errors.Is(err, ErrExit) {
		t.Fatalf("err = %v, want ErrExit", err)
	}
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v, want *ExitError", err)
	}
	if ee.ExitCode != 1 {
		t.Fatalf("ExitError.ExitCode = %d, want 1", ee.ExitCode)
	}
	if res.ExitCode != 1 {
		t.Fatalf("Result.ExitCode = %d, want 1 (result must still be returned)", res.ExitCode)
	}
}

func TestRun_StderrOnFailure(t *testing.T) {
	t.Parallel()
	res, err := Exec{}.Run(withTimeout(t), "ls", "/definitely-not-a-path-execx")
	if !errors.Is(err, ErrExit) {
		t.Fatalf("err = %v, want ErrExit", err)
	}
	if res.Stderr == "" {
		t.Fatalf("stderr empty, want ls error text")
	}
	if res.ExitCode == 0 {
		t.Fatalf("ExitCode = 0, want non-zero")
	}
}

func TestRun_NotFound(t *testing.T) {
	t.Parallel()
	_, err := Exec{}.Run(withTimeout(t), "definitely-not-a-binary-execx-xyz")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("err = %v, want wrapping exec.ErrNotFound", err)
	}
}

func TestRun_CancelKills(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := Exec{}.Run(ctx, "sleep", "30")
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel; process was not killed")
	}
}

func TestRun_DeadlineExceeded(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := Exec{}.Run(ctx, "sleep", "30")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestRun_CancelKillsProcessTree(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := Exec{}.Run(ctx, "/bin/sh", "-c", "/bin/sleep 41 & wait")
		done <- err
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel; child likely held stdout/stderr")
	}

	out, err := exec.Command("ps", "-eo", "args=").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		args := strings.TrimSpace(line)
		if args == "/bin/sleep 41" {
			t.Fatalf("child still alive: %q", args)
		}
	}
}
