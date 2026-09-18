package apk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func (t *Tools) Pull(ctx context.Context, serial, pkg string, layout workspace.Layout) (Artifact, error) {
	if serial == "" || pkg == "" {
		return Artifact{}, fmt.Errorf("apk: pull: %w", ErrUsage)
	}
	if err := layout.Ensure(); err != nil {
		return Artifact{}, fmt.Errorf("apk: pull: %w", err)
	}
	res, err := t.r.Run(ctx, "adb", "-s", serial, "shell", "pm", "path", pkg)
	if errors.Is(err, execx.ErrNotFound) {
		return Artifact{}, fmt.Errorf("apk: pull: %w: adb", ErrToolMissing)
	}
	if errors.Is(err, execx.ErrExit) {
		return Artifact{}, fmt.Errorf("apk: pull: %w", ErrPackageNotFound)
	}
	if err != nil {
		return Artifact{}, fmt.Errorf("apk: pull: %w", err)
	}
	remote, splits, err := parsePMPath(res.Stdout)
	if err != nil {
		return Artifact{}, fmt.Errorf("apk: pull: %w", err)
	}
	dest := filepath.Join(layout.APK, "base.apk")
	_, err = t.r.Run(ctx, "adb", "-s", serial, "pull", remote, dest)
	if errors.Is(err, execx.ErrNotFound) {
		return Artifact{}, fmt.Errorf("apk: pull: %w: adb", ErrToolMissing)
	}
	if err != nil {
		return Artifact{}, fmt.Errorf("apk: pull: %w", err)
	}
	return Artifact{Package: pkg, APK: dest, SkippedSplits: splits}, nil
}

func (t *Tools) CopyAPK(_ context.Context, src, pkg string, layout workspace.Layout) (Artifact, error) {
	if src == "" || pkg == "" {
		return Artifact{}, fmt.Errorf("apk: copy: %w", ErrUsage)
	}
	if err := layout.Ensure(); err != nil {
		return Artifact{}, fmt.Errorf("apk: copy: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return Artifact{}, fmt.Errorf("apk: copy: %w", err)
	}
	dest := filepath.Join(layout.APK, "base.apk")
	out, err := os.Create(dest)
	if err != nil {
		_ = in.Close()
		return Artifact{}, fmt.Errorf("apk: copy: %w", err)
	}
	_, copyErr := io.Copy(out, in)
	closeOut := out.Close()
	closeIn := in.Close()
	if copyErr != nil {
		return Artifact{}, fmt.Errorf("apk: copy: %w", copyErr)
	}
	if closeOut != nil {
		return Artifact{}, fmt.Errorf("apk: copy: %w", closeOut)
	}
	if closeIn != nil {
		return Artifact{}, fmt.Errorf("apk: copy: %w", closeIn)
	}
	return Artifact{Package: pkg, APK: dest}, nil
}
