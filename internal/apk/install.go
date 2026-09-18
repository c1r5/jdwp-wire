package apk

import (
	"context"
	"fmt"
)

func (t *Tools) Install(ctx context.Context, serial, apkPath string) error {
	if serial == "" || apkPath == "" {
		return fmt.Errorf("apk: install: %w", ErrUsage)
	}
	res, err := t.r.Run(ctx, "adb", "-s", serial, "install", "-r", "-d", apkPath)
	if err != nil {
		return wrapRun("install", "adb", res.Stderr, err)
	}
	return nil
}
