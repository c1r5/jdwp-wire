package attach

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/androidcli"
	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/install"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/patchapply"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/c1r5/jdwp-wire/internal/pull"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// StudioStatus is what attach did with the IntelliJ skeleton.
type StudioStatus int

const (
	StudioOff StudioStatus = iota
	StudioWritten
	StudioNoDecode
)

// Launch is which path started the process.
type Launch string

const (
	LaunchADB     Launch = "adb"
	LaunchAndroid Launch = "android"
)

type studioWriter interface {
	Write(cfg project.Config) (project.Result, error)
}

var (
	_ studioWriter = project.FS{}
	_ studioWriter = (*project.Fake)(nil)
)

// Options is the attach request. Writer is used only when Studio is set.
type Options struct {
	Serial  string
	Package string
	Port    int
	CWD     string
	Studio  bool
	Writer  studioWriter
	APK     apk.Client
	Patch   patch.Applier
	Android androidcli.Client
}

// Result is the JDWP session plus whether the app was repackaged.
type Result struct {
	Session    jdwp.Session
	Studio     StudioStatus
	Project    project.Result
	Repackaged bool
	Launch     Launch
	SkipDecode bool
	PullAPK    string
	DecodeDir  string
	Patch      patch.Result
	Signed     string
	Installed  bool
}

type repack struct {
	pull       string
	decode     string
	skipDecode bool
	patch      patch.Result
	apks       []string
}

// Run pulls and decodes, patches what is still missing, installs, forwards JDWP,
// and with Studio writes the IntelliJ skeleton. A step that is already done is
// skipped on its own; later steps still run.
func Run(ctx context.Context, dev device.Client, client jdwp.Client, opt Options) (Result, error) {
	layout, err := workspace.ForPackage(opt.CWD, opt.Package)
	if err != nil {
		return Result{}, fmt.Errorf("attach: %v: %w", err, jdwp.ErrUsage)
	}
	if opt.Studio && opt.Writer == nil {
		return Result{}, fmt.Errorf("attach: studio: %w", project.ErrUsage)
	}

	d, err := dev.Resolve(ctx, opt.Serial)
	if err != nil {
		return Result{}, err
	}
	packed, err := prepare(ctx, dev, opt, d.Serial, layout)
	if err != nil {
		return Result{}, err
	}

	sess, launch, installed, err := start(ctx, client, opt, d.Serial, &packed)
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Session:    sess,
		Launch:     launch,
		Installed:  installed,
		Repackaged: true,
		SkipDecode: packed.skipDecode,
		PullAPK:    packed.pull,
		DecodeDir:  packed.decode,
		Patch:      packed.patch,
	}
	if len(packed.apks) > 0 {
		res.Signed = packed.apks[0]
	}
	if !opt.Studio {
		return res, nil
	}
	wrote, err := opt.Writer.Write(project.Config{Layout: layout, Port: sess.Port})
	if errors.Is(err, project.ErrNoDecode) {
		res.Studio = StudioNoDecode
		return res, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("attach: studio: %w", err)
	}
	res.Studio = StudioWritten
	res.Project = wrote
	return res, nil
}

func prepare(ctx context.Context, dev device.Client, opt Options, serial string, layout workspace.Layout) (repack, error) {
	if opt.APK == nil || opt.Patch == nil {
		return repack{}, fmt.Errorf("attach: prepare: apk and patch clients are required")
	}
	out := repack{decode: layout.Decode}
	if decodeReady(layout.Decode) {
		out.skipDecode = true
		out.pull = filepath.Join(layout.APK, "base.apk")
	} else {
		pulled, err := pull.Run(ctx, pull.Deps{Device: dev, APK: opt.APK, CWD: opt.CWD}, pull.Request{
			Target: opt.Package,
			Serial: serial,
			Decode: true,
		})
		if err != nil {
			return repack{}, err
		}
		out.pull = pulled.Artifact.APK
		out.decode = pulled.DecodeDir
	}
	patched, err := patchapply.Run(ctx, patchapply.Deps{APK: opt.APK, Patch: opt.Patch, CWD: opt.CWD}, patchapply.Request{
		DecodeDir: out.decode,
	})
	if err != nil {
		return repack{}, err
	}
	out.patch = patched
	signed, err := install.Run(ctx, install.Deps{Device: dev, APK: opt.APK, CWD: opt.CWD}, install.Request{
		Target:   out.decode,
		Serial:   serial,
		Package:  opt.Package,
		SignOnly: true,
	})
	if err != nil {
		return repack{}, err
	}
	out.apks = append([]string{signed.APK}, signed.Splits...)
	return out, nil
}

func decodeReady(dir string) bool {
	for _, name := range []string{"AndroidManifest.xml", "apktool.yml"} {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.IsDir() {
			return false
		}
	}
	return true
}

func start(ctx context.Context, client jdwp.Client, opt Options, serial string, packed *repack) (jdwp.Session, Launch, bool, error) {
	ok, err := androidAvailable(ctx, opt.Android)
	if err != nil {
		return jdwp.Session{}, "", false, err
	}
	if ok {
		if err := deploy(ctx, opt, serial, packed.apks, true); err != nil {
			return jdwp.Session{}, "", false, err
		}
		sess, err := client.Bind(ctx, serial, opt.Package, opt.Port)
		return sess, LaunchAndroid, false, err
	}
	if err := deploy(ctx, opt, serial, packed.apks, false); err != nil {
		return jdwp.Session{}, "", false, err
	}
	sess, err := client.Attach(ctx, serial, opt.Package, opt.Port)
	return sess, LaunchADB, true, err
}

func deploy(ctx context.Context, opt Options, serial string, apks []string, useAndroid bool) error {
	err := installOnce(ctx, opt, serial, apks, useAndroid)
	if !signatureMismatch(err) {
		return err
	}
	if err := opt.APK.Uninstall(ctx, serial, opt.Package); err != nil {
		return err
	}
	return installOnce(ctx, opt, serial, apks, useAndroid)
}

func installOnce(ctx context.Context, opt Options, serial string, apks []string, useAndroid bool) error {
	if useAndroid {
		return opt.Android.RunDebug(ctx, serial, apks)
	}
	return opt.APK.Install(ctx, serial, apks...)
}

func signatureMismatch(err error) bool {
	return err != nil && strings.Contains(err.Error(), "INSTALL_FAILED_UPDATE_INCOMPATIBLE")
}

func androidAvailable(ctx context.Context, c androidcli.Client) (bool, error) {
	if c == nil {
		return false, nil
	}
	return c.Available(ctx)
}
