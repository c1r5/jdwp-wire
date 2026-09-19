package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
)

func TestAttachHuman(t *testing.T) {
	t.Parallel()
	var gotSerial, gotPkg string
	var gotPort int
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
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
		"[ok] debug-app: com.alvo",
		"[ok] launch: com.alvo/.MainActivity",
		"[ok] jdwp: pid 4242",
		"[ok] forward: tcp:8700 -> jdwp:4242",
		"[ok] probe: 127.0.0.1:8700",
		"[skip] patch: use jdt patch and jdt install first",
		"[skip] studio: not wired",
		"attach: localhost:8700",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestAttachMonkeyLaunchLine(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
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
	if !strings.Contains(stdout.String(), "[ok] launch: monkey com.alvo") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestAttachJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
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
		Package  string `json:"package"`
		Serial   string `json:"serial"`
		PID      int    `json:"pid"`
		Port     int    `json:"port"`
		Activity string `json:"activity"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Package != "com.alvo" || got.Serial != "emulator-5554" || got.PID != 4242 || got.Port != 9000 || got.Activity != "com.alvo/.MainActivity" {
		t.Fatalf("%+v", got)
	}
}

func TestAttachNoDevice(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
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
	code := run(runConfig{
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
	code := run(runConfig{
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

func TestAttachStudioFlagAccepted(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		jdwp: &jdwp.Fake{AttachFn: func(_ context.Context, serial, pkg string, port int) (jdwp.Session, error) {
			return jdwp.Session{Package: pkg, Serial: serial, PID: 1, Port: port, Activity: "x/.Y"}, nil
		}},
		args: []string{"attach", "com.alvo", "--studio"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
}

func TestResetHuman(t *testing.T) {
	t.Parallel()
	var gotSerial, gotPkg string
	var gotPort int
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
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
	if !strings.Contains(out, "[ok] reset: clear-debug-app com.alvo") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "[ok] reset: remove forward tcp:8700") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestResetJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
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
	code := run(runConfig{
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
