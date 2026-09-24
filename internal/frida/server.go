package frida

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

const (
	serverRemote = "/data/local/tmp/frida-server"
	serverLaunch = "su -c '/data/local/tmp/frida-server && sleep 2147483647 &'"
	serverBoot   = time.Second
)

// Server is the frida-server process on the device.
type Server struct {
	PID     int
	Started bool
}

// EnsureServer checks pidof frida-server and, if it is down, starts the binary
// already at /data/local/tmp/frida-server. The one-second adb timeout is the
// detach hack: it is not an error by itself. The second pidof decides.
func EnsureServer(ctx context.Context, run execx.Runner, serial string) (Server, error) {
	if run == nil || serial == "" {
		return Server{}, fmt.Errorf("frida: server: %w", ErrUsage)
	}
	pid, err := pidof(ctx, run, serial)
	if err != nil {
		return Server{}, err
	}
	if pid > 0 {
		return Server{PID: pid}, nil
	}
	if _, err := run.Run(ctx, "adb", "-s", serial, "shell", "ls", serverRemote); err != nil {
		if errors.Is(err, execx.ErrNotFound) {
			return Server{}, fmt.Errorf("frida: server: %w", ErrToolMissing)
		}
		return Server{}, fmt.Errorf("frida: server: %s missing", serverRemote)
	}
	boot, cancel := context.WithTimeout(ctx, serverBoot)
	_, launchErr := run.Run(boot, "adb", "-s", serial, "shell", serverLaunch)
	cancel()
	if launchErr != nil && !errors.Is(launchErr, context.DeadlineExceeded) && !errors.Is(launchErr, context.Canceled) {
		// su can fail in the same second the server still comes up. pidof decides.
		pid, err = pidof(ctx, run, serial)
		if err != nil {
			return Server{}, err
		}
		if pid > 0 {
			return Server{PID: pid, Started: true}, nil
		}
		return Server{}, fmt.Errorf("frida: server: %w", launchErr)
	}
	pid, err = pidof(ctx, run, serial)
	if err != nil {
		return Server{}, err
	}
	if pid <= 0 {
		return Server{}, fmt.Errorf("frida: server: did not start")
	}
	return Server{PID: pid, Started: true}, nil
}

func pidof(ctx context.Context, run execx.Runner, serial string) (int, error) {
	res, err := run.Run(ctx, "adb", "-s", serial, "shell", "pidof", "frida-server")
	if errors.Is(err, execx.ErrNotFound) {
		return 0, fmt.Errorf("frida: server: %w", ErrToolMissing)
	}
	fields := strings.Fields(res.Stdout)
	if len(fields) == 0 {
		if err != nil && !errors.Is(err, execx.ErrExit) {
			return 0, fmt.Errorf("frida: server: pidof: %w", err)
		}
		return 0, nil
	}
	n, conv := strconv.Atoi(fields[0])
	if conv != nil || n <= 0 {
		return 0, nil
	}
	return n, nil
}
