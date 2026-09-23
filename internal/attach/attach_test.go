package attach

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func testDevice() device.Client {
	return &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
	}}}
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
