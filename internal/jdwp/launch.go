package jdwp

import (
	"context"
	"errors"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func (a *ADB) setDebugApp(ctx context.Context, serial, pkg string) error {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "am", "set-debug-app", "-w", pkg)
	if err != nil {
		return wrapRun("attach", "adb", res.Stderr, err)
	}
	return nil
}

func (a *ADB) launch(ctx context.Context, serial, pkg string) (string, error) {
	res, err := a.r.Run(ctx, "adb", "-s", serial, "shell", "cmd", "package", "resolve-activity",
		"--brief", "-c", "android.intent.category.LAUNCHER", pkg)
	if err != nil && !errors.Is(err, execx.ErrExit) {
		return "", wrapRun("launch", "adb", res.Stderr, err)
	}
	if act := parseActivity(res.Stdout); act != "" {
		res, err = a.r.Run(ctx, "adb", "-s", serial, "shell", "am", "start", "-D", "-n", act)
		if err != nil {
			return "", wrapRun("launch", "adb", res.Stderr, err)
		}
		return act, nil
	}
	res, err = a.r.Run(ctx, "adb", "-s", serial, "shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
	if err != nil {
		return "", wrapRun("launch", "adb", res.Stderr, err)
	}
	return "", nil
}
