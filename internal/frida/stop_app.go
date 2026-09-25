package frida

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

// stopApp force-stops the package, clears the debug app, and removes the
// JDWP forward. It does not stop frida-server. A nil runner uses execx.
func stopApp(run execx.Runner, serial, pkg string, port int) error {
	if serial == "" || pkg == "" {
		return nil
	}
	if run == nil {
		run = execx.Exec{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var first error
	keep := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}
	_, err := run.Run(ctx, "adb", "-s", serial, "shell", "am", "force-stop", pkg)
	keep(err)
	_, err = run.Run(ctx, "adb", "-s", serial, "shell", "am", "clear-debug-app")
	keep(err)
	if port >= 1 && port <= 65535 {
		_, err = run.Run(ctx, "adb", "-s", serial, "forward", "--remove", "tcp:"+strconv.Itoa(port))
		if err != nil && !errors.Is(err, execx.ErrExit) {
			keep(err)
		}
	}
	if first != nil {
		return fmt.Errorf("frida: stop: %w", first)
	}
	return nil
}
