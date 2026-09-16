package apk

import (
	"context"
	"errors"

	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

var (
	ErrToolMissing     = errors.New("tool missing")
	ErrPackageNotFound = errors.New("package not installed")
	ErrNoBaseAPK       = errors.New("no base.apk")
	ErrUsage           = errors.New("usage")
)

type Artifact struct {
	Package       string
	APK           string
	SkippedSplits []string
}

type Decoded struct {
	Package string
	Dir     string
}

// Client pulls, decodes, signs, and installs APKs.
// Temporary: move to the consumer when that shape stabilizes.
type Client interface {
	Pull(ctx context.Context, serial, pkg string, layout workspace.Layout) (Artifact, error)
	CopyAPK(ctx context.Context, src, pkg string, layout workspace.Layout) (Artifact, error)
	Decode(ctx context.Context, apkPath string, layout workspace.Layout) (Decoded, error)
	Build(ctx context.Context, decodedDir, outAPK string) (Artifact, error)
	Sign(ctx context.Context, apkPath, keystore string) (Artifact, error)
	Install(ctx context.Context, serial, apkPath string) error
}

type Tools struct {
	r    execx.Runner
	look func(string) (string, error)
}

func New(r execx.Runner) *Tools {
	return &Tools{r: r, look: execx.Look}
}
