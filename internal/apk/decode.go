package apk

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func (t *Tools) Decode(ctx context.Context, apkPath string, layout workspace.Layout) (Decoded, error) {
	if apkPath == "" || layout.Decode == "" {
		return Decoded{}, fmt.Errorf("apk: decode: %w", ErrUsage)
	}
	res, err := t.r.Run(ctx, "apktool", "d", "-f", "-o", layout.Decode, apkPath)
	if err != nil {
		return Decoded{}, wrapRun("decode", "apktool", res.Stderr, err)
	}
	return Decoded{
		Package: filepath.Base(layout.Root),
		Dir:     layout.Decode,
	}, nil
}

func (t *Tools) Build(ctx context.Context, decodedDir, outAPK string) (Artifact, error) {
	if decodedDir == "" || outAPK == "" {
		return Artifact{}, fmt.Errorf("apk: build: %w", ErrUsage)
	}
	res, err := t.r.Run(ctx, "apktool", "b", "-o", outAPK, decodedDir)
	if err != nil {
		return Artifact{}, wrapRun("build", "apktool", res.Stderr, err)
	}
	return Artifact{APK: outAPK}, nil
}
