package attach

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/androidcli"
	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func testDevice() device.Client {
	return &device.Fake{
		Devices: []device.Device{{
			Serial: "emulator-5554",
			State:  device.StateDevice,
			Kind:   device.KindEmulator,
		}},
		DebugPackages: map[string]bool{"com.alvo": true},
	}
}

func TestRun(t *testing.T) {
	t.Parallel()
	called := false
	sess, err := Run(context.Background(), testDevice(), &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
		return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port}, nil
	}}, Options{
		Package: "com.alvo",
		Port:    8700,
		Writer: &project.Fake{WriteFn: func(project.Config) (project.Result, error) {
			called = true
			return project.Result{}, nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("writer called without --studio")
	}
	if sess.Studio != StudioOff {
		t.Fatalf("studio %d", sess.Studio)
	}
	if sess.Session.Serial != "emulator-5554" || sess.Session.Package != "com.alvo" || sess.Session.Port != 8700 {
		t.Fatalf("%+v", sess.Session)
	}
}

func TestRunStudioWrites(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	want, err := workspace.ForPackage(cwd, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	var got project.Config
	res, err := Run(context.Background(), testDevice(), &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
		return jdwp.Session{Package: pkg, Serial: serial, PID: 7, Port: port}, nil
	}}, Options{
		Package: "com.alvo",
		Port:    9000,
		CWD:     cwd,
		Studio:  true,
		Writer: &project.Fake{WriteFn: func(cfg project.Config) (project.Result, error) {
			got = cfg
			return project.Result{Dir: cfg.Layout.Idea}, nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Studio != StudioWritten || res.Project.Dir != want.Idea {
		t.Fatalf("%+v", res)
	}
	if got.Port != 9000 || got.Layout.Idea != want.Idea || got.Layout.Decode != want.Decode {
		t.Fatalf("cfg %+v", got)
	}
}

func TestRunStudioNoDecode(t *testing.T) {
	t.Parallel()
	res, err := Run(context.Background(), testDevice(), &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
		return jdwp.Session{Package: pkg, Serial: serial, PID: 3, Port: port}, nil
	}}, Options{
		Package: "com.alvo",
		Port:    8700,
		CWD:     t.TempDir(),
		Studio:  true,
		Writer: &project.Fake{WriteFn: func(project.Config) (project.Result, error) {
			return project.Result{}, project.ErrNoDecode
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Studio != StudioNoDecode || res.Session.PID != 3 {
		t.Fatalf("%+v", res)
	}
}

func TestRunStudioWriteError(t *testing.T) {
	t.Parallel()
	_, err := Run(context.Background(), testDevice(), &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
		return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port}, nil
	}}, Options{
		Package: "com.alvo",
		Port:    8700,
		CWD:     t.TempDir(),
		Studio:  true,
		Writer: &project.Fake{WriteFn: func(project.Config) (project.Result, error) {
			return project.Result{}, errors.New("disk")
		}},
	})
	if err == nil || errors.Is(err, project.ErrNoDecode) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunStudioBadPackage(t *testing.T) {
	t.Parallel()
	_, err := Run(context.Background(), testDevice(), &jdwp.Fake{AttachFn: func(context.Context, string, string, int) (jdwp.Session, error) {
		t.Fatal("attach called")
		return jdwp.Session{}, nil
	}}, Options{
		Package: filepath.Join("com.alvo", "extra"),
		Port:    8700,
		CWD:     t.TempDir(),
		Studio:  true,
		Writer:  project.FS{},
	})
	if !errors.Is(err, jdwp.ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}

func releaseDevice() *device.Fake {
	return &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
	}}}
}

func repackAPK(t *testing.T) *apk.Fake {
	t.Helper()
	return &apk.Fake{
		PullFn: func(_ context.Context, _, pkg string, _ workspace.Layout) (apk.Artifact, error) {
			return apk.Artifact{Package: pkg, APK: "/tmp/" + pkg + "/base.apk"}, nil
		},
		DecodeFn: func(_ context.Context, _ string, layout workspace.Layout) (apk.Decoded, error) {
			if err := os.MkdirAll(layout.Decode, 0o755); err != nil {
				return apk.Decoded{}, err
			}
			if err := os.WriteFile(filepath.Join(layout.Decode, "AndroidManifest.xml"), []byte("<manifest/>"), 0o644); err != nil {
				return apk.Decoded{}, err
			}
			if err := os.WriteFile(filepath.Join(layout.Decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
				return apk.Decoded{}, err
			}
			return apk.Decoded{Package: "com.alvo", Dir: layout.Decode}, nil
		},
		BuildFn: func(_ context.Context, _, out string) (apk.Artifact, error) {
			return apk.Artifact{APK: out}, nil
		},
		SignFn: func(_ context.Context, path, _ string) (apk.Artifact, error) {
			return apk.Artifact{APK: path + ".signed"}, nil
		},
		InstallFn: func(context.Context, string, ...string) error {
			t.Fatal("adb install")
			return nil
		},
	}
}

func appliedPatch() *patch.Fake {
	return &patch.Fake{ApplyFn: func(dir string) (patch.Result, error) {
		return patch.Result{Dir: dir, Debuggable: patch.ActionApplied, NSC: patch.ActionApplied}, nil
	}}
}

func TestRunRepackageAndroid(t *testing.T) {
	t.Parallel()
	var launched []string
	var bound bool
	tools := repackAPK(t)
	res, err := Run(context.Background(), releaseDevice(), &jdwp.Fake{
		AttachFn: func(context.Context, string, string, int) (jdwp.Session, error) {
			t.Fatal("adb attach")
			return jdwp.Session{}, nil
		},
		BindFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			bound = true
			return jdwp.Session{Package: pkg, Serial: serial, PID: 9, Port: port}, nil
		},
	}, Options{
		Package: "com.alvo",
		Port:    8700,
		CWD:     t.TempDir(),
		APK:     tools,
		Patch:   appliedPatch(),
		Android: &androidcli.Fake{
			AvailableFn: func(context.Context) (bool, error) { return true, nil },
			RunDebugFn: func(_ context.Context, serial string, apks []string) error {
				if serial != "emulator-5554" || len(apks) != 1 {
					t.Fatalf("run %s %v", serial, apks)
				}
				launched = apks
				return nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bound || !res.Repackaged || res.Launch != LaunchAndroid || res.Installed || res.Session.PID != 9 {
		t.Fatalf("%+v bound=%v", res, bound)
	}
	if len(launched) != 1 || launched[0] != res.Signed {
		t.Fatalf("launched %v signed %s", launched, res.Signed)
	}
	if res.Patch.Debuggable != patch.ActionApplied || res.DecodeDir == "" || res.PullAPK == "" {
		t.Fatalf("%+v", res)
	}
}

func TestRunRepackageFallsBackToADB(t *testing.T) {
	t.Parallel()
	var installed []string
	var attached bool
	tools := repackAPK(t)
	tools.InstallFn = func(_ context.Context, serial string, apks ...string) error {
		if serial != "emulator-5554" {
			t.Fatalf("serial %s", serial)
		}
		installed = apks
		return nil
	}
	res, err := Run(context.Background(), releaseDevice(), &jdwp.Fake{
		AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			attached = true
			return jdwp.Session{Package: pkg, Serial: serial, PID: 4, Port: port, Activity: "com.alvo/.Main"}, nil
		},
		BindFn: func(context.Context, string, string, int) (jdwp.Session, error) {
			t.Fatal("bind")
			return jdwp.Session{}, nil
		},
	}, Options{
		Package: "com.alvo",
		Port:    8700,
		CWD:     t.TempDir(),
		APK:     tools,
		Patch:   appliedPatch(),
		Android: &androidcli.Fake{AvailableFn: func(context.Context) (bool, error) { return false, nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !attached || !res.Repackaged || res.Launch != LaunchADB || !res.Installed || len(installed) != 1 {
		t.Fatalf("%+v installed %v", res, installed)
	}
}

func TestRunMissingPackage(t *testing.T) {
	t.Parallel()
	dev := releaseDevice()
	dev.Missing = map[string]bool{"com.alvo": true}
	_, err := Run(context.Background(), dev, &jdwp.Fake{
		AttachFn: func(context.Context, string, string, int) (jdwp.Session, error) {
			t.Fatal("attach")
			return jdwp.Session{}, nil
		},
	}, Options{Package: "com.alvo", Port: 8700, CWD: t.TempDir()})
	if !errors.Is(err, device.ErrPackageNotFound) {
		t.Fatalf("err=%v", err)
	}
}
