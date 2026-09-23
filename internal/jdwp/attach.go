package jdwp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func validate(serial, pkg string, port int) error {
	if serial == "" || pkg == "" || port < 1 || port > 65535 {
		return ErrUsage
	}
	return nil
}

func (a *ADB) Attach(ctx context.Context, serial, pkg string, port int) (Session, error) {
	if err := validate(serial, pkg, port); err != nil {
		return Session{}, fmt.Errorf("jdwp: attach: %w", err)
	}
	if err := a.setDebugApp(ctx, serial, pkg); err != nil {
		return Session{}, err
	}
	a.log.OK("debug-app", pkg)
	act, err := a.launch(ctx, serial, pkg)
	if err != nil {
		return Session{}, err
	}
	if act != "" {
		a.log.OK("launch", act)
	} else {
		a.log.OK("launch", "monkey "+pkg)
	}
	pid, err := a.waitPID(ctx, serial, pkg)
	if err != nil {
		return Session{}, err
	}
	a.log.OK("jdwp", fmt.Sprintf("pid %d", pid))
	if err := a.forward(ctx, serial, port, pid); err != nil {
		return Session{}, err
	}
	if err := a.confirmForward(ctx, serial, port, pid); err != nil {
		return Session{}, err
	}
	a.logForward(serial, port, pid)
	return Session{Package: pkg, Serial: serial, PID: pid, Port: port, Activity: act}, nil
}

func (a *ADB) Bind(ctx context.Context, serial, pkg string, port int) (Session, error) {
	if err := validate(serial, pkg, port); err != nil {
		return Session{}, fmt.Errorf("jdwp: bind: %w", err)
	}
	pid, err := a.waitPID(ctx, serial, pkg)
	if err != nil {
		return Session{}, err
	}
	a.log.OK("jdwp", fmt.Sprintf("pid %d", pid))
	if err := a.forward(ctx, serial, port, pid); err != nil {
		return Session{}, err
	}
	if err := a.confirmForward(ctx, serial, port, pid); err != nil {
		return Session{}, err
	}
	a.logForward(serial, port, pid)
	return Session{Package: pkg, Serial: serial, PID: pid, Port: port}, nil
}

func (a *ADB) Reset(ctx context.Context, serial, pkg string, port int) error {
	if err := validate(serial, pkg, port); err != nil {
		return fmt.Errorf("jdwp: reset: %w", err)
	}
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "am", "clear-debug-app")
	if err != nil {
		return wrapRun("reset", "adb", res.Stderr, err)
	}
	a.log.OK("reset", "clear-debug-app "+pkg)
	res, err = a.r.Run(ctx, "adb", "-s", serial, "forward", "--remove", "tcp:"+strconv.Itoa(port))
	if err != nil && !errors.Is(err, execx.ErrExit) {
		return wrapRun("reset", "adb", res.Stderr, err)
	}
	a.log.OK("reset", fmt.Sprintf("remove forward tcp:%d", port))
	return nil
}

func (a *ADB) logForward(serial string, port, pid int) {
	a.log.OK("forward", fmt.Sprintf("adb -s %s tcp:%d -> jdwp:%d", serial, port, pid))
	a.log.Skip("probe", "jdwp socket left for the debugger")
}

func (a *ADB) forward(ctx context.Context, serial string, port, pid int) error {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "forward",
		"tcp:"+strconv.Itoa(port), "jdwp:"+strconv.Itoa(pid))
	if err != nil {
		return wrapRun("forward", "adb", res.Stderr, err)
	}
	return nil
}

func (a *ADB) confirmForward(ctx context.Context, serial string, port, pid int) error {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "forward", "--list")
	if err != nil {
		return wrapRun("forward", "adb", res.Stderr, err)
	}
	want := fmt.Sprintf("%s tcp:%d jdwp:%d", serial, port, pid)
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.TrimSpace(line) == want {
			return nil
		}
	}
	return fmt.Errorf("jdwp: forward: %w", ErrProbe)
}
