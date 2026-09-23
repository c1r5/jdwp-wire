package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/logging"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// Deps is the device/apk workspace wiring for jdt install.
type Deps struct {
	Device device.Client
	APK    apk.Client
	CWD    string
	// Log is written as each step finishes. Nil stays quiet.
	Log *logging.Logger
}

// Request is the parsed input for jdt install.
// SignOnly builds and signs and does not call adb install.
type Request struct {
	Target   string
	Serial   string
	Package  string
	SignOnly bool
}

// Result is a signed APK set installed on a device.
type Result struct {
	APK     string
	Splits  []string
	Serial  string
	Encoded bool
}

type prep struct {
	Base    string
	Splits  []string
	Encoded bool
}

// Run rebuilds a decode tree if needed, signs debug, and adb-installs.
func Run(ctx context.Context, deps Deps, req Request) (Result, error) {
	dev, err := deps.Device.Resolve(ctx, req.Serial)
	if err != nil {
		return Result{}, err
	}
	p, err := prepare(ctx, deps, req.Target, req.Package)
	if err != nil {
		return Result{}, err
	}
	apks := append([]string{p.Base}, p.Splits...)
	ks := workspace.DebugKeystore(deps.CWD)
	for i, path := range apks {
		signed, err := deps.APK.Sign(ctx, path, ks)
		if err != nil {
			return Result{}, err
		}
		if signed.APK != "" {
			apks[i] = signed.APK
		}
		deps.Log.OK("sign", apks[i])
	}
	if !req.SignOnly {
		if err := deps.APK.Install(ctx, dev.Serial, apks...); err != nil {
			return Result{}, err
		}
		for _, p := range apks {
			deps.Log.OK("install", p)
		}
	}
	out := Result{APK: apks[0], Serial: dev.Serial, Encoded: p.Encoded}
	if len(apks) > 1 {
		out.Splits = apks[1:]
	}
	return out, nil
}

func prepare(ctx context.Context, deps Deps, target, pkgFlag string) (prep, error) {
	st, err := os.Stat(target)
	if err == nil && !st.IsDir() {
		return prep{Base: target}, nil
	}
	pkg := pkgFlag
	decodeDir := ""
	if err == nil && st.IsDir() {
		if !isDecodedTree(target) {
			return prep{}, fmt.Errorf("%w: not an apktool decode directory", apk.ErrUsage)
		}
		if pkg == "" {
			pkg, err = packageFromDecodeDir(deps.CWD, target)
			if err != nil {
				return prep{}, err
			}
		}
		decodeDir = target
	} else {
		pkg = target
		layout, err := workspace.ForPackage(deps.CWD, pkg)
		if err != nil {
			return prep{}, err
		}
		if !isDecodedTree(layout.Decode) {
			return prep{}, fmt.Errorf("install: decode: %w", os.ErrNotExist)
		}
		decodeDir = layout.Decode
	}
	base, encoded, err := buildPatched(ctx, deps, decodeDir, pkg)
	if err != nil {
		return prep{}, err
	}
	splits, err := stageSplits(deps.CWD, pkg)
	if err != nil {
		return prep{}, err
	}
	return prep{Base: base, Splits: splits, Encoded: encoded}, nil
}

func stageSplits(cwd, pkg string) ([]string, error) {
	layout, err := workspace.ForPackage(cwd, pkg)
	if err != nil {
		return nil, err
	}
	src, err := apk.ListSplits(layout.APK)
	if err != nil {
		return nil, err
	}
	dests := make([]string, 0, len(src))
	for _, s := range src {
		dst := filepath.Join(layout.Patched, filepath.Base(s))
		if err := copyFile(s, dst); err != nil {
			return nil, err
		}
		dests = append(dests, dst)
	}
	return dests, nil
}

func copyFile(src, dst string) error {
	if src == dst {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		_ = in.Close()
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		_ = in.Close()
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeOut := out.Close()
	closeIn := in.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeOut != nil {
		return closeOut
	}
	return closeIn
}

func buildPatched(ctx context.Context, deps Deps, decodedDir, pkg string) (string, bool, error) {
	layout, err := workspace.ForPackage(deps.CWD, pkg)
	if err != nil {
		return "", false, err
	}
	if err := layout.Ensure(); err != nil {
		return "", false, err
	}
	out := filepath.Join(layout.Patched, "base.apk")
	art, err := deps.APK.Build(ctx, decodedDir, out)
	if err != nil {
		return "", false, err
	}
	if art.APK != "" {
		out = art.APK
	}
	deps.Log.OK("encode", out)
	return out, true, nil
}

func isDecodedTree(dir string) bool {
	for _, name := range []string{"apktool.yml", "AndroidManifest.xml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func packageFromDecodeDir(cwd, dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(filepath.Join(absCwd, ".jdt"), absDir)
	if err != nil {
		return "", fmt.Errorf("%w: --package required for decode dir", apk.ErrUsage)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("%w: --package required for decode dir", apk.ErrUsage)
	}
	parts := strings.Split(rel, "/")
	if len(parts) != 2 || parts[1] != "decode" {
		return "", fmt.Errorf("%w: --package required for decode dir", apk.ErrUsage)
	}
	return parts[0], nil
}
