package apk

import (
	"context"
	"fmt"
)

func (t *Tools) Install(ctx context.Context, serial string, apks ...string) error {
	if serial == "" || len(apks) == 0 {
		return fmt.Errorf("apk: install: %w", ErrUsage)
	}
	for _, p := range apks {
		if p == "" {
			return fmt.Errorf("apk: install: %w", ErrUsage)
		}
	}
	args := []string{"-s", serial}
	if len(apks) == 1 {
		args = append(args, "install", "-r", "-d", apks[0])
	} else {
		args = append(args, "install-multiple", "-r", "-d")
		args = append(args, apks...)
	}
	res, err := t.r.Run(ctx, "adb", args...)
	if err != nil {
		return wrapRun("install", "adb", res.Stderr, err)
	}
	return nil
}
