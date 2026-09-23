package execx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Cmd is a running external process with its stdio pipes. Stdin stays open
// until Wait or Kill so a tool that exits on EOF keeps running.
type Cmd struct {
	cmd    *exec.Cmd
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	stdin  io.WriteCloser
}

// PID is the operating system process id. It is 0 before the process exists.
func (c *Cmd) PID() int {
	if c == nil || c.cmd == nil || c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}

// Wait waits for the process to exit. Read Stdout and Stderr first.
// Stdin stays open so a CLI that exits on EOF is not stopped by Wait itself.
func (c *Cmd) Wait() error {
	if c == nil || c.cmd == nil {
		return fmt.Errorf("execx: wait: %w", ErrExit)
	}
	name := c.cmd.Path
	err := c.cmd.Wait()
	c.closeIn()
	if err == nil || (errors.Is(err, exec.ErrWaitDelay) && c.cmd.ProcessState != nil && c.cmd.ProcessState.Success()) {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("execx: %s: %w", name, err)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return fmt.Errorf("execx: %s: %w", name, &ExitError{Name: name, ExitCode: ee.ExitCode()})
	}
	return fmt.Errorf("execx: %s: %w", name, err)
}

// Kill stops the process group on Unix and the process elsewhere.
func (c *Cmd) Kill() error {
	if c == nil || c.cmd == nil {
		return nil
	}
	c.closeIn()
	if c.cmd.Cancel != nil {
		return c.cmd.Cancel()
	}
	if c.cmd.Process == nil {
		return nil
	}
	return c.cmd.Process.Kill()
}

func (c *Cmd) closeIn() {
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
}

// Start runs name without waiting. Unlike Run, a context without a deadline is
// allowed: cancel still kills the process. The caller reads Stdout and Stderr
// and then Wait.
func (Exec) Start(ctx context.Context, name string, args ...string) (*Cmd, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execx: %s: %w", name, ErrNoDeadline)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	configureKill(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("execx: %s: %w", name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("execx: %s: %w", name, err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("execx: %s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("execx: %s: %w", name, fmt.Errorf("%w: %w", ErrNotFound, exec.ErrNotFound))
		}
		return nil, fmt.Errorf("execx: %s: %w", name, err)
	}
	return &Cmd{cmd: cmd, Stdout: stdout, Stderr: stderr, stdin: stdin}, nil
}
