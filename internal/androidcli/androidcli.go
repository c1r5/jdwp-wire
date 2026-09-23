package androidcli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

const maxErrBody = 2048

var (
	ErrUsage       = errors.New("usage")
	ErrToolMissing = errors.New("tool missing")
)

// Client detects the official Android CLI and deploys with run --debug.
type Client interface {
	Available(ctx context.Context) (bool, error)
	RunDebug(ctx context.Context, serial string, apks []string) error
}

// CLI shells out to the android binary.
type CLI struct {
	r    execx.Runner
	look func(string) (string, error)
}

func New(r execx.Runner) *CLI {
	return &CLI{r: r, look: execx.Look}
}

// Available reports whether android is the 1.0 CLI (android run --apks/--debug).
// A missing binary and the legacy SDK android tool are both false.
func (c *CLI) Available(ctx context.Context) (bool, error) {
	if _, err := c.look("android"); err != nil {
		if errors.Is(err, execx.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("androidcli: available: %w", err)
	}
	res, err := c.r.Run(ctx, "android", "run", "-h")
	if err != nil && !errors.Is(err, execx.ErrExit) {
		if errors.Is(err, execx.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("androidcli: available: %w", err)
	}
	text := res.Stdout + res.Stderr
	if strings.Contains(text, "--apks") && strings.Contains(text, "--debug") {
		return true, nil
	}
	return false, nil
}

// RunDebug installs and launches apks with android run --debug.
func (c *CLI) RunDebug(ctx context.Context, serial string, apks []string) error {
	if serial == "" || len(apks) == 0 {
		return fmt.Errorf("androidcli: run: %w", ErrUsage)
	}
	if _, err := c.look("android"); err != nil {
		if errors.Is(err, execx.ErrNotFound) {
			return fmt.Errorf("androidcli: run: %w: android", ErrToolMissing)
		}
		return fmt.Errorf("androidcli: run: %w", err)
	}
	res, err := c.r.Run(ctx, "android",
		"run", "--debug",
		"--device="+serial,
		"--apks="+strings.Join(apks, ","),
		"--install-options=-r,-d",
	)
	if err != nil {
		return wrapRun(res.Stderr, err)
	}
	return nil
}

func wrapRun(stderr string, err error) error {
	if errors.Is(err, execx.ErrNotFound) {
		return fmt.Errorf("androidcli: run: %w: android", ErrToolMissing)
	}
	if errors.Is(err, execx.ErrExit) {
		exit := &execx.ExitError{Name: "android", ExitCode: 1, Stderr: stderr}
		var orig *execx.ExitError
		if errors.As(err, &orig) {
			exit.ExitCode = orig.ExitCode
			if exit.Stderr == "" {
				exit.Stderr = orig.Stderr
			}
		}
		exit.Stderr = truncate(exit.Stderr, maxErrBody)
		return fmt.Errorf("androidcli: run: %w", exit)
	}
	return fmt.Errorf("androidcli: run: %w", err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
