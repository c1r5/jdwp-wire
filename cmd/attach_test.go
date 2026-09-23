package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/androidcli"
	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func testAPK(t *testing.T) *apk.Fake {
	t.Helper()
	return &apk.Fake{
		PullFn: func(_ context.Context, _, pkg string, _ workspace.Layout) (apk.Artifact, error) {
			return apk.Artifact{Package: pkg, APK: "/apk/base.apk"}, nil
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
			return apk.Decoded{Dir: layout.Decode, Package: "com.alvo"}, nil
		},
		BuildFn: func(_ context.Context, _, out string) (apk.Artifact, error) {
			return apk.Artifact{APK: out}, nil
		},
		SignFn: func(_ context.Context, path, _ string) (apk.Artifact, error) {
			return apk.Artifact{APK: path}, nil
		},
		InstallFn:   func(context.Context, string, ...string) error { return nil },
		UninstallFn: func(context.Context, string, string) error { return nil },
	}
}

func attachRun(t *testing.T, cfg runConfig) int {
	t.Helper()
	if cfg.cwd == "" {
		cfg.cwd = t.TempDir()
	}
	if cfg.apk == nil {
		cfg.apk = testAPK(t)
	}
	if cfg.patch == nil {
		cfg.patch = &patch.Fake{ApplyFn: func(dir string) (patch.Result, error) {
			return patch.Result{Dir: dir, Debuggable: patch.ActionSkipped, NSC: patch.ActionSkipped}, nil
		}}
	}
	if cfg.android == nil {
		cfg.android = &androidcli.Fake{}
	}
	return run(cfg)
}

func TestAttachHuman(t *testing.T) {
	t.Parallel()
	var gotSerial, gotPkg string
	var gotPort int
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			gotSerial, gotPkg, gotPort = serial, pkg, port
			return jdwp.Session{
				Package:  pkg,
				Serial:   serial,
				PID:      4242,
				Port:     port,
				Activity: "com.alvo/.MainActivity",
			}, nil
		}},
		args: []string{"attach", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if gotSerial != "emulator-5554" || gotPkg != "com.alvo" || gotPort != 8700 {
		t.Fatalf("attach %s %s %d", gotSerial, gotPkg, gotPort)
	}
	out := stdout.String()
	for _, want := range []string{
		"[ok] [pull] /apk/base.apk",
		"[ok] [decode] ",
		"[skip] [patch] already debuggable",
		"[skip] [patch] nsc already trusts user CA",
		"[ok] [sign] ",
		"[skip] [android] cli unavailable",
		"[ok] [install] ",
		"[ok] [debug-app] com.alvo",
		"[ok] [launch] com.alvo/.MainActivity",
		"[ok] [jdwp] pid 4242",
		"[ok] [device] emulator emulator-5554 phone",
		"[ok] [forward] adb -s emulator-5554 tcp:8700 -> jdwp:4242",
		"[skip] [probe] jdwp socket left for the debugger",
		"[skip] [studio] pass --studio",
		"[ok] [attach] 127.0.0.1:8700",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestAttachRepackageAndroid(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    cwd,
		device: &device.Fake{Devices: []device.Device{{
			Serial: "emulator-5554",
			State:  device.StateDevice,
		}}},
		apk: &apk.Fake{
			PullFn: func(_ context.Context, _, pkg string, _ workspace.Layout) (apk.Artifact, error) {
				return apk.Artifact{Package: pkg, APK: "/apk/base.apk"}, nil
			},
			DecodeFn: func(_ context.Context, _ string, layout workspace.Layout) (apk.Decoded, error) {
				if err := os.MkdirAll(layout.Decode, 0o755); err != nil {
					return apk.Decoded{}, err
				}
				if err := os.WriteFile(filepath.Join(layout.Decode, "apktool.yml"), []byte("version: 2\n"), 0o644); err != nil {
					return apk.Decoded{}, err
				}
				return apk.Decoded{Dir: layout.Decode, Package: "com.alvo"}, nil
			},
			BuildFn: func(_ context.Context, _, out string) (apk.Artifact, error) {
				return apk.Artifact{APK: out}, nil
			},
			SignFn: func(_ context.Context, path, _ string) (apk.Artifact, error) {
				return apk.Artifact{APK: path}, nil
			},
		},
		patch: &patch.Fake{ApplyFn: func(dir string) (patch.Result, error) {
			return patch.Result{Dir: dir, Debuggable: patch.ActionApplied, NSC: patch.ActionApplied}, nil
		}},
		android: &androidcli.Fake{
			AvailableFn: func(context.Context) (bool, error) { return true, nil },
			RunDebugFn:  func(context.Context, string, []string) error { return nil },
		},
		jdwp: &jdwp.Fake{BindFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 8, Port: port}, nil
		}},
		args: []string{"attach", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"[ok] [pull] /apk/base.apk",
		"[ok] [patch] debuggable",
		"[ok] [patch] nsc user CA",
		"[ok] [launch] android run --debug",
		"[ok] [jdwp] pid 8",
		"[ok] [attach] 127.0.0.1:8700",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "debug-app") {
		t.Fatalf("adb launch in android path:\n%s", out)
	}
}

func TestAttachMonkeyLaunchLine(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 9, Port: port}, nil
		}},
		args: []string{"attach", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[ok] [launch] monkey com.alvo") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestAttachJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{
				Package:  pkg,
				Serial:   serial,
				PID:      4242,
				Port:     port,
				Activity: "com.alvo/.MainActivity",
			}, nil
		}},
		args: []string{"attach", "com.alvo", "--json", "--port", "9000"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Package    string `json:"package"`
		Serial     string `json:"serial"`
		PID        int    `json:"pid"`
		Port       int    `json:"port"`
		Activity   string `json:"activity"`
		Repackaged bool   `json:"repackaged"`
		Launch     string `json:"launch"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Package != "com.alvo" || got.Serial != "emulator-5554" || got.PID != 4242 || got.Port != 9000 || got.Activity != "com.alvo/.MainActivity" || !got.Repackaged || got.Launch != "adb" {
		t.Fatalf("%+v", got)
	}
	if strings.Contains(stdout.String(), "studio") {
		t.Fatalf("studio field without flag:\n%s", stdout.String())
	}
}

func TestAttachNoDevice(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: &device.Fake{},
		jdwp:   &jdwp.Fake{},
		args:   []string{"attach", "com.alvo"},
	})
	if code != ExitNoDevice {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitNoDevice, stderr.String())
	}
}

func TestAttachNoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp:   &jdwp.Fake{},
		args:   []string{"attach"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestAttachToolMissing(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, _, _ string, _ int) (jdwp.Session, error) {
			return jdwp.Session{}, jdwp.ErrToolMissing
		}},
		args: []string{"attach", "com.alvo"},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitToolMissing, stderr.String())
	}
}

func TestAttachStudioOffDoesNotWrite(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port}, nil
		}},
		projects: &project.Fake{WriteFn: func(project.Config) (project.Result, error) {
			t.Fatal("writer called")
			return project.Result{}, nil
		}},
		args: []string{"attach", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[skip] [studio] pass --studio") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestAttachStudioWrites(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	decode := filepath.Join(cwd, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "AndroidManifest.xml"), []byte("<manifest/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    cwd,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port, Activity: "com.alvo/.Main"}, nil
		}},
		args: []string{"attach", "com.alvo", "--studio", "--port", "9000"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	idea := filepath.Join(cwd, ".jdt", "com.alvo", "idea")
	if !strings.Contains(stdout.String(), "[ok] [studio] "+idea+"\n") {
		t.Fatalf("stdout=%q", stdout.String())
	}
	for _, name := range []string{
		"decode.iml",
		filepath.Join(".idea", "misc.xml"),
		filepath.Join(".idea", "modules.xml"),
		filepath.Join(".idea", "runConfigurations", "Remote_Debug.xml"),
	} {
		if _, err := os.Stat(filepath.Join(idea, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAttachStudioNoDecode(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    cwd,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port}, nil
		}},
		args: []string{"attach", "com.alvo", "--studio"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	idea := filepath.Join(cwd, ".jdt", "com.alvo", "idea")
	if !strings.Contains(stdout.String(), "[ok] [studio] "+idea+"\n") {
		t.Fatalf("stdout=%q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(idea, "decode.iml")); err != nil {
		t.Fatal(err)
	}
}

func TestAttachStudioJSON(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	decode := filepath.Join(cwd, ".jdt", "com.alvo", "decode")
	if err := os.MkdirAll(decode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decode, "AndroidManifest.xml"), []byte("<manifest/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    cwd,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 4, Port: port}, nil
		}},
		args: []string{"attach", "com.alvo", "--studio", "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Port   int `json:"port"`
		Studio struct {
			Dir     string `json:"dir"`
			Skipped bool   `json:"skipped"`
			Reason  string `json:"reason"`
		} `json:"studio"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	idea := filepath.Join(cwd, ".jdt", "com.alvo", "idea")
	if got.Port != 8700 || got.Studio.Skipped || got.Studio.Reason != "" || got.Studio.Dir != idea {
		t.Fatalf("%+v", got)
	}
}

func TestAttachStudioJSONNoDecode(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    cwd,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 4, Port: port}, nil
		}},
		args: []string{"attach", "com.alvo", "--studio", "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Studio struct {
			Dir     string `json:"dir"`
			Skipped bool   `json:"skipped"`
			Reason  string `json:"reason"`
		} `json:"studio"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	idea := filepath.Join(cwd, ".jdt", "com.alvo", "idea")
	if got.Studio.Skipped || got.Studio.Reason != "" || got.Studio.Dir != idea {
		t.Fatalf("%+v", got)
	}
}

func TestAttachStudioBadPackage(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		cwd:    t.TempDir(),
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(context.Context, string, string, int) (jdwp.Session, error) {
			t.Fatal("attach called")
			return jdwp.Session{}, nil
		}},
		args: []string{"attach", "com.alvo/extra", "--studio"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestResetHuman(t *testing.T) {
	t.Parallel()
	var gotSerial, gotPkg string
	var gotPort int
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{ResetFn: func(_ context.Context, serial, pkg string, port int) error {
			gotSerial, gotPkg, gotPort = serial, pkg, port
			return nil
		}},
		args: []string{"reset", "com.alvo"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if gotSerial != "emulator-5554" || gotPkg != "com.alvo" || gotPort != 8700 {
		t.Fatalf("reset %s %s %d", gotSerial, gotPkg, gotPort)
	}
	out := stdout.String()
	if !strings.Contains(out, "[ok] [reset] clear-debug-app com.alvo") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[ok] [reset] remove forward tcp:8700") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestResetJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{ResetFn: func(_ context.Context, _, _ string, _ int) error {
			return nil
		}},
		args: []string{"reset", "com.alvo", "--json", "-s", "emulator-5554"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	var got struct {
		Package string `json:"package"`
		Serial  string `json:"serial"`
		Port    int    `json:"port"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Package != "com.alvo" || got.Serial != "emulator-5554" || got.Port != 8700 {
		t.Fatalf("%+v", got)
	}
}

func TestResetNoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := attachRun(t, runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp:   &jdwp.Fake{},
		args:   []string{"reset"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitUsage, stderr.String())
	}
}
