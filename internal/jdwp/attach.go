package jdwp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

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
	act, err := a.launch(ctx, serial, pkg)
	if err != nil {
		return Session{}, err
	}
	pid, err := a.waitPID(ctx, serial, pkg)
	if err != nil {
		return Session{}, err
	}
	if err := a.forward(ctx, serial, port, pid); err != nil {
		return Session{}, err
	}
	if err := a.probe(ctx, port); err != nil {
		return Session{}, err
	}
	return Session{Package: pkg, Serial: serial, PID: pid, Port: port, Activity: act}, nil
}

func (a *ADB) Reset(ctx context.Context, serial, pkg string, port int) error {
	if err := validate(serial, pkg, port); err != nil {
		return fmt.Errorf("jdwp: reset: %w", err)
	}
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "am", "clear-debug-app")
	if err != nil {
		return wrapRun("reset", "adb", res.Stderr, err)
	}
	res, err = a.r.Run(ctx, "adb", "-s", serial, "forward", "--remove", "tcp:"+strconv.Itoa(port))
	if errors.Is(err, execx.ErrExit) {
		return nil
	}
	if err != nil {
		return wrapRun("reset", "adb", res.Stderr, err)
	}
	return nil
}

func (a *ADB) forward(ctx context.Context, serial string, port, pid int) error {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "forward",
		"tcp:"+strconv.Itoa(port), "jdwp:"+strconv.Itoa(pid))
	if err != nil {
		return wrapRun("forward", "adb", res.Stderr, err)
	}
	return nil
}

func (a *ADB) probe(ctx context.Context, port int) error {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("jdwp: probe: %w", ErrProbe)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("jdwp: probe: %w", ErrProbe)
	}
	return nil
}
