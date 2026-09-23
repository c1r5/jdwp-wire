package jdwp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func (a *ADB) waitPID(ctx context.Context, serial, pkg string) (int, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("jdwp: wait: %w", ErrNoProcess)
		}
		procs, err := a.d.Pidof(ctx, serial, pkg)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			if errors.Is(err, execx.ErrExit) {
				procs = nil
			} else {
				return 0, fmt.Errorf("jdwp: wait: %w", err)
			}
		}
		if pid := pickPID(pkg, procs); pid > 0 {
			ok, err := a.jdwpHas(ctx, serial, pid)
			if err != nil {
				return 0, err
			}
			if ok {
				return pid, nil
			}
		}
		if err := a.sleep(ctx); err != nil {
			return 0, fmt.Errorf("jdwp: wait: %w", ErrNoProcess)
		}
	}
}

func (a *ADB) jdwpHas(ctx context.Context, serial string, pid int) (bool, error) {
	// adb jdwp prints the pid list and then stays open. Bound the read so the
	// parent attach context is not spent waiting for a process that never exits.
	wait := a.listFor
	if wait <= 0 {
		wait = 2 * time.Second
	}
	listCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	res, err := a.r.Run(listCtx, "adb", "-s", serial, "jdwp")
	if ctx.Err() != nil {
		return false, fmt.Errorf("jdwp: wait: %w", ctx.Err())
	}
	if err != nil && !errors.Is(err, execx.ErrExit) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		return false, wrapRun("wait", "adb", res.Stderr, err)
	}
	for _, n := range parseJDWP(res.Stdout) {
		if n == pid {
			return true, nil
		}
	}
	return false, nil
}

func (a *ADB) sleep(ctx context.Context) error {
	d := a.poll
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
