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
	"github.com/c1r5/jdwp-wire/internal/frida"
	"github.com/c1r5/jdwp-wire/internal/install"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/logging"
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
	// Log receives each step as it finishes. Nil stays quiet (--json).
	Log *logging.Logger
	// Refs loads Frida after the forward. Empty skips Frida entirely.
	Refs   []frida.Ref
	Detach bool
	Frida  frida.Deps
	// Lifetime is canceled on SIGINT/SIGTERM and not by the command timeout.
	// The CLI calls Stop when it ends so the app is closed before the process exits.
	Lifetime context.Context
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
	Device     device.Device
	Frida      *frida.Opened
	// Launched is set once install/launch has started, including when a later step fails.
	Launched bool
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
	// Install can take long enough for the cable to drop. Forward only if this
	// serial is still the connected device.
	live, err := dev.Resolve(ctx, d.Serial)
	if err != nil {
		return Result{}, err
	}
	opt.Log.OK("device", formatDevice(live))

	selfLog := false
	if s, ok := client.(interface{ SetLog(*logging.Logger) }); ok {
		s.SetLog(opt.Log)
		selfLog = opt.Log != nil
	}
	res := Result{
		Repackaged: true,
		SkipDecode: packed.skipDecode,
		PullAPK:    packed.pull,
		DecodeDir:  packed.decode,
		Patch:      packed.patch,
		Device:     live,
	}
	sess, launch, installed, err := start(ctx, client, opt, live.Serial, &packed, selfLog, &res)
	if err != nil {
		return res, err
	}
	res.Session = sess
	res.Launch = launch
	res.Installed = installed
	if len(packed.apks) > 0 {
		res.Signed = packed.apks[0]
	}
	opened, err := armFrida(ctx, opt, live.Serial, sess.PID, layout)
	res.Frida = opened
	if err != nil {
		return res, err
	}
	if !opt.Studio {
		res.Studio = StudioOff
		opt.Log.Skip("studio", "pass --studio")
		logAttach(opt.Log, sess.Port)
		return res, nil
	}
	wrote, err := opt.Writer.Write(project.Config{Layout: layout, Port: sess.Port})
	if errors.Is(err, project.ErrNoDecode) {
		res.Studio = StudioNoDecode
		opt.Log.Skip("studio", "no decode")
		logAttach(opt.Log, sess.Port)
		return res, nil
	}
	if err != nil {
		return res, fmt.Errorf("attach: studio: %w", err)
	}
	res.Studio = StudioWritten
	res.Project = wrote
	opt.Log.OK("studio", wrote.Dir)
	logAttach(opt.Log, sess.Port)
	return res, nil
}

func formatDevice(d device.Device) string {
	parts := make([]string, 0, 3)
	if d.Kind != "" {
		parts = append(parts, string(d.Kind))
	}
	if d.Serial != "" {
		parts = append(parts, d.Serial)
	}
	if d.Model != "" {
		parts = append(parts, d.Model)
	}
	return strings.Join(parts, " ")
}

func logAttach(lg *logging.Logger, port int) {
	lg.OK("attach", fmt.Sprintf("127.0.0.1:%d", port))
}

func armFrida(ctx context.Context, opt Options, serial string, pid int, layout workspace.Layout) (*frida.Opened, error) {
	if len(opt.Refs) == 0 {
		return nil, nil
	}
	dir := filepath.Join(layout.Root, "frida")
	dep := opt.Frida
	if dep.ProcCtx == nil {
		dep.ProcCtx = opt.Lifetime
	}
	opened, srv, err := frida.Open(ctx, dep, serial, opt.Package, pid, opt.Port, dir, opt.Refs, opt.Detach)
	if err != nil {
		return nil, err
	}
	if srv.Started {
		opt.Log.OK("frida", "server started")
	} else {
		opt.Log.Skip("frida", "server already running")
	}
	opt.Log.OK("frida", strings.Join(opened.Names, ", "))
	opt.Log.OK("frida", displayPath(opt.CWD, opened.Log))
	return &opened, nil
}

func displayPath(cwd, path string) string {
	rel, err := filepath.Rel(cwd, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

func prepare(ctx context.Context, dev device.Client, opt Options, serial string, layout workspace.Layout) (repack, error) {
	if opt.APK == nil || opt.Patch == nil {
		return repack{}, fmt.Errorf("attach: prepare: apk and patch clients are required")
	}
	out := repack{decode: layout.Decode}
	if decodeReady(layout.Decode) {
		out.skipDecode = true
		out.pull = filepath.Join(layout.APK, "base.apk")
		opt.Log.Skip("pull", "decode already exists")
		opt.Log.Skip("decode", out.decode)
	} else {
		pulled, err := pull.Run(ctx, pull.Deps{Device: dev, APK: opt.APK, CWD: opt.CWD, Log: opt.Log}, pull.Request{
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
	patched, err := patchapply.Run(ctx, patchapply.Deps{APK: opt.APK, Patch: opt.Patch, CWD: opt.CWD, Log: opt.Log}, patchapply.Request{
		DecodeDir: out.decode,
	})
	if err != nil {
		return repack{}, err
	}
	out.patch = patched
	signed, err := install.Run(ctx, install.Deps{Device: dev, APK: opt.APK, CWD: opt.CWD, Log: opt.Log}, install.Request{
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

func start(ctx context.Context, client jdwp.Client, opt Options, serial string, packed *repack, selfLog bool, res *Result) (jdwp.Session, Launch, bool, error) {
	ok, err := androidAvailable(ctx, opt.Android)
	if err != nil {
		return jdwp.Session{}, "", false, err
	}
	res.Launched = true
	res.Session.Package = opt.Package
	res.Session.Serial = serial
	res.Session.Port = opt.Port
	if ok {
		if err := deploy(ctx, opt, serial, packed.apks, true); err != nil {
			return jdwp.Session{}, "", false, err
		}
		opt.Log.OK("launch", "android run --debug")
		sess, err := client.Bind(ctx, serial, opt.Package, opt.Port)
		if err != nil {
			return jdwp.Session{}, "", false, err
		}
		if !selfLog {
			logBound(opt.Log, sess)
		}
		return sess, LaunchAndroid, false, nil
	}
	opt.Log.Skip("android", "cli unavailable")
	if err := deploy(ctx, opt, serial, packed.apks, false); err != nil {
		return jdwp.Session{}, "", false, err
	}
	for _, p := range packed.apks {
		opt.Log.OK("install", p)
	}
	sess, err := client.Attach(ctx, serial, opt.Package, opt.Port)
	if err != nil {
		return jdwp.Session{}, "", false, err
	}
	if !selfLog {
		logADBSession(opt.Log, sess)
	}
	return sess, LaunchADB, true, nil
}

func logADBSession(lg *logging.Logger, sess jdwp.Session) {
	lg.OK("debug-app", sess.Package)
	if sess.Activity != "" {
		lg.OK("launch", sess.Activity)
	} else {
		lg.OK("launch", "monkey "+sess.Package)
	}
	logBound(lg, sess)
}

func logBound(lg *logging.Logger, sess jdwp.Session) {
	lg.OK("jdwp", fmt.Sprintf("pid %d", sess.PID))
	lg.OK("forward", fmt.Sprintf("adb -s %s tcp:%d -> jdwp:%d", sess.Serial, sess.Port, sess.PID))
	lg.Skip("probe", "jdwp socket left for the debugger")
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
