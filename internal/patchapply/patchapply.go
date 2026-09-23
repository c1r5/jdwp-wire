package patchapply

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/logging"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// Deps is the apk/patch wiring for jdt patch.
type Deps struct {
	APK        apk.Client
	Patch      patch.Applier
	CWD        string
	APKTimeout time.Duration
	// Log is written as each step finishes. Nil stays quiet.
	Log *logging.Logger
}

// Request is the parsed input for jdt patch.
type Request struct {
	DecodeDir string
	APK       string
	Package   string
}

// Run decodes if needed and applies the debuggable + NSC patch.
func Run(ctx context.Context, deps Deps, req Request) (patch.Result, error) {
	if req.APK != "" && req.DecodeDir != "" {
		return patch.Result{}, fmt.Errorf("%w: decoded_dir and --apk are mutually exclusive", patch.ErrUsage)
	}
	if req.APK != "" {
		if req.Package == "" {
			return patch.Result{}, fmt.Errorf("%w: --package required for --apk", patch.ErrUsage)
		}
		if !fileExists(req.APK) {
			return patch.Result{}, fmt.Errorf("patch: apk: %w", os.ErrNotExist)
		}
		layout, err := workspace.ForPackage(deps.CWD, req.Package)
		if err != nil {
			return patch.Result{}, err
		}
		if deps.APKTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, deps.APKTimeout)
			defer cancel()
		}
		dec, err := deps.APK.Decode(ctx, req.APK, layout)
		if err != nil {
			return patch.Result{}, err
		}
		deps.Log.OK("decode", dec.Dir)
		return applyAndLog(deps, dec.Dir)
	}
	if req.DecodeDir == "" {
		return patch.Result{}, fmt.Errorf("%w: decoded_dir or --apk required", patch.ErrUsage)
	}
	st, err := os.Stat(req.DecodeDir)
	if err != nil {
		return patch.Result{}, err
	}
	if !st.IsDir() {
		return patch.Result{}, fmt.Errorf("%w: decoded_dir must be a directory", patch.ErrUsage)
	}
	return applyAndLog(deps, req.DecodeDir)
}

func applyAndLog(deps Deps, dir string) (patch.Result, error) {
	res, err := deps.Patch.Apply(dir)
	if err != nil {
		return patch.Result{}, err
	}
	if res.Debuggable == patch.ActionApplied {
		deps.Log.OK("patch", "debuggable")
	} else {
		deps.Log.Skip("patch", "already debuggable")
	}
	if res.NSC == patch.ActionApplied {
		deps.Log.OK("patch", "nsc user CA")
	} else {
		deps.Log.Skip("patch", "nsc already trusts user CA")
	}
	return res, nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
