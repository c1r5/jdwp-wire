package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrNoDeadline = errors.New("no deadline")
	ErrNotFound   = errors.New("not found")
	ErrExit       = errors.New("exit")
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner executes an external binary. Temporary: move to the consumer
// (cli/jdwp) when that shape stabilizes. Fake shared across modules
// justifies keeping it here for the MVP.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

type ExitError struct {
	Name     string
	ExitCode int
	Stderr   string
}

func (e *ExitError) Error() string {
	if e == nil {
		return "exit"
	}
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		return fmt.Sprintf("exit %d", e.ExitCode)
	}
	return fmt.Sprintf("exit %d: %s", e.ExitCode, msg)
}

func (e *ExitError) Unwrap() error { return ErrExit }

func Look(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("execx: %s: %w", name, fmt.Errorf("%w: %w", ErrNotFound, exec.ErrNotFound))
	}
	return path, nil
}

type Exec struct{}

func (Exec) Run(ctx context.Context, name string, args ...string) (Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		return Result{}, fmt.Errorf("execx: %s: %w", name, ErrNoDeadline)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	configureKill(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}
	// A descendant that inherits the pipes makes Wait return ErrWaitDelay after
	// the process itself has already exited 0. The captured output is complete.
	if err == nil || (errors.Is(err, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success()) {
		return res, nil
	}

	if isNotFound(err) {
		return res, fmt.Errorf("execx: %s: %w", name, fmt.Errorf("%w: %w", ErrNotFound, exec.ErrNotFound))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return res, fmt.Errorf("execx: %s: %w", name, ctxErr)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return res, fmt.Errorf("execx: %s: %w", name, &ExitError{
			Name:     name,
			ExitCode: ee.ExitCode(),
			Stderr:   res.Stderr,
		})
	}
	return res, fmt.Errorf("execx: %s: %w", name, err)
}

func isNotFound(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
		return true
	}
	var execErr *exec.Error
	return errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)
}
