package attach

import (
	"context"
	"errors"
	"fmt"
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
	PullAPK    string
	DecodeDir  string
	Patch      patch.Result
	Signed     string
	Installed  bool
}

type repack struct {
	pull   string
	decode string
	patch  patch.Result
	apks   []string
}

// Run resolves the device, repackages when the app is not debuggable, forwards JDWP,
// and with Studio writes the IntelliJ skeleton.
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
	debuggable, err := dev.Debuggable(ctx, d.Serial, opt.Package)
	if err != nil {
		return Result{}, err
	}

	var packed *repack
	if !debuggable {
		got, err := repackage(ctx, dev, opt, d.Serial)
		if err != nil {
			return Result{}, err
		}
		packed = &got
	}

	sess, launch, installed, err := start(ctx, client, opt, d.Serial, packed)
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Session:   sess,
		Launch:    launch,
		Installed: installed,
	}
	if packed != nil {
		res.Repackaged = true
		res.PullAPK = packed.pull
		res.DecodeDir = packed.decode
		res.Patch = packed.patch
		if len(packed.apks) > 0 {
			res.Signed = packed.apks[0]
		}
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

func repackage(ctx context.Context, dev device.Client, opt Options, serial string) (repack, error) {
	if opt.APK == nil || opt.Patch == nil {
		return repack{}, fmt.Errorf("attach: repackage: apk and patch clients are required")
	}
	pulled, err := pull.Run(ctx, pull.Deps{Device: dev, APK: opt.APK, CWD: opt.CWD}, pull.Request{
		Target: opt.Package,
		Serial: serial,
		Decode: true,
	})
	if err != nil {
		return repack{}, err
	}
	patched, err := patchapply.Run(ctx, patchapply.Deps{APK: opt.APK, Patch: opt.Patch, CWD: opt.CWD}, patchapply.Request{
		DecodeDir: pulled.DecodeDir,
	})
	if err != nil {
		return repack{}, err
	}
	signed, err := install.Run(ctx, install.Deps{Device: dev, APK: opt.APK, CWD: opt.CWD}, install.Request{
		Target:   pulled.DecodeDir,
		Serial:   serial,
		Package:  opt.Package,
		SignOnly: true,
	})
	if err != nil {
		return repack{}, err
	}
	apks := append([]string{signed.APK}, signed.Splits...)
	return repack{pull: pulled.Artifact.APK, decode: pulled.DecodeDir, patch: patched, apks: apks}, nil
}

func start(ctx context.Context, client jdwp.Client, opt Options, serial string, packed *repack) (jdwp.Session, Launch, bool, error) {
	if packed == nil {
		sess, err := client.Attach(ctx, serial, opt.Package, opt.Port)
		return sess, LaunchADB, false, err
	}
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
