package execx

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestStart_Stdout(t *testing.T) {
	t.Parallel()
	cmd, err := Exec{}.Start(context.Background(), "/bin/echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(cmd.Stderr); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Fatalf("stdout = %q", out)
	}
	if cmd.PID() == 0 {
		t.Fatal("pid")
	}
}

func TestStart_CancelKills(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cmd, err := Exec{}.Start(ctx, "sleep", "30")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("sleep exited 0 after cancel")
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Kill()
		t.Fatal("sleep still running after cancel")
	}
}

func TestStart_NotFound(t *testing.T) {
	t.Parallel()
	_, err := Exec{}.Start(context.Background(), "definitely-not-a-binary-execx-xyz")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("err = %v, want exec.ErrNotFound", err)
	}
}
