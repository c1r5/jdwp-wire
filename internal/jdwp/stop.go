package jdwp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

const stopTimeout = 8 * time.Second

// ForceStop closes the app and the debug session. The command context is
// already canceled on SIGTERM, so this call times out on its own.
func (a *ADB) ForceStop(serial, pkg string, port int) error {
	if a == nil || serial == "" || pkg == "" {
		return fmt.Errorf("jdwp: stop: %w", ErrUsage)
	}
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	var first error
	keep := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "am", "force-stop", pkg)
	if err != nil {
		keep(wrapRun("stop", "adb", res.Stderr, err))
	} else {
		a.log.OK("stop", "force-stop "+pkg)
	}
	res, err = a.r.Run(ctx, "adb", "-s", serial, "shell", "am", "clear-debug-app")
	if err != nil {
		keep(wrapRun("stop", "adb", res.Stderr, err))
	} else {
		a.log.OK("stop", "clear-debug-app "+pkg)
	}
	if port < 1 || port > 65535 {
		return first
	}
	res, err = a.r.Run(ctx, "adb", "-s", serial, "forward", "--remove", "tcp:"+strconv.Itoa(port))
	if err != nil && !errors.Is(err, execx.ErrExit) {
		keep(wrapRun("stop", "adb", res.Stderr, err))
	} else {
		a.log.OK("stop", fmt.Sprintf("remove forward tcp:%d", port))
	}
	return first
}
