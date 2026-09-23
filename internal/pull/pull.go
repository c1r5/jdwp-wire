package pull

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/apps"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/logging"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// Deps is the device/apk workspace wiring for jdt pull.
type Deps struct {
	Device device.Client
	APK    apk.Client
	CWD    string
	// Log is written as each step finishes. Nil stays quiet.
	Log *logging.Logger
}

// Request is the parsed input for jdt pull.
type Request struct {
	Target  string
	Serial  string
	Package string
	Decode  bool
	// System selects the jdt apps --system index space.
	System bool
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
	logPull(deps.Log, art)
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
	deps.Log.OK("decode", dec.Dir)
	out.DecodeDir = dec.Dir
	return out, nil
}

func logPull(lg *logging.Logger, art apk.Artifact) {
	lg.OK("pull", art.Package+" → "+art.APK)
	for _, s := range art.Splits {
		lg.OK("pull", s)
	}
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
	pkg := req.Target
	if isAppIndex(pkg) {
		pkg, err = packageByIndex(ctx, deps, dev.Serial, pkg, req.System)
		if err != nil {
			return apk.Artifact{}, err
		}
	}
	layout, err := workspace.ForPackage(deps.CWD, pkg)
	if err != nil {
		return apk.Artifact{}, err
	}
	return deps.APK.Pull(ctx, dev.Serial, pkg, layout)
}

func packageByIndex(ctx context.Context, deps Deps, serial, raw string, includeSystem bool) (string, error) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return "", fmt.Errorf("%w: app index %s", device.ErrUsage, raw)
	}
	entries, err := apps.List(ctx, deps.Device, serial, includeSystem)
	if err != nil {
		return "", err
	}
	return apps.PackageAt(entries, n)
}

func isAppIndex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
