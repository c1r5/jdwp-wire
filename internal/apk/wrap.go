package apk

import (
	"errors"
	"fmt"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

const maxErrBody = 2048

func wrapRun(op, tool string, stderr string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, execx.ErrNotFound) {
		return fmt.Errorf("apk: %s: %w: %s", op, ErrToolMissing, tool)
	}
	if errors.Is(err, execx.ErrExit) {
		exit := &execx.ExitError{Name: tool, ExitCode: 1, Stderr: stderr}
		var orig *execx.ExitError
		if errors.As(err, &orig) {
			exit.ExitCode = orig.ExitCode
			if exit.Stderr == "" {
				exit.Stderr = orig.Stderr
			}
		}
		exit.Stderr = truncate(exit.Stderr, maxErrBody)
		return fmt.Errorf("apk: %s: %w", op, exit)
	}
	return fmt.Errorf("apk: %s: %w", op, err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
