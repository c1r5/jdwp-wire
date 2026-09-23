package pull

import (
	"context"
	"fmt"
	"os"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// Deps is the device/apk workspace wiring for jdt pull.
type Deps struct {
	Device device.Client
	APK    apk.Client
	CWD    string
}

// Request is the parsed input for jdt pull.
type Request struct {
	Target  string
	Serial  string
	Package string
	Decode  bool
}

// Result is a pulled (and optionally decoded) artifact.
type Result struct {
	Artifact  apk.Artifact
	DecodeDir string
}

// Run pulls or copies an APK into .jdt/ and optionally apktool-decodes it.
func Run(ctx context.Context, deps Deps, req Request) (Result, error) {
	art, err := fetch(ctx, deps, req)
	if err != nil {
		return Result{}, err
	}
	out := Result{Artifact: art}
	if !req.Decode {
		return out, nil
	}
	layout, err := workspace.ForPackage(deps.CWD, art.Package)
	if err != nil {
		return Result{}, err
	}
	dec, err := deps.APK.Decode(ctx, art.APK, layout)
	if err != nil {
		return Result{}, err
	}
	out.DecodeDir = dec.Dir
	return out, nil
}

func fetch(ctx context.Context, deps Deps, req Request) (apk.Artifact, error) {
	if fileExists(req.Target) {
		if req.Package == "" {
			return apk.Artifact{}, fmt.Errorf("%w: --package required for local apk", apk.ErrUsage)
		}
		layout, err := workspace.ForPackage(deps.CWD, req.Package)
		if err != nil {
			return apk.Artifact{}, err
		}
		return deps.APK.CopyAPK(ctx, req.Target, req.Package, layout)
	}
	dev, err := deps.Device.Resolve(ctx, req.Serial)
	if err != nil {
		return apk.Artifact{}, err
	}
	layout, err := workspace.ForPackage(deps.CWD, req.Target)
	if err != nil {
		return apk.Artifact{}, err
	}
	return deps.APK.Pull(ctx, dev.Serial, req.Target, layout)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
