package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestDevicesHumanTable(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
		Model:  "sdk_gphone64_x86_64",
	}}}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"devices"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr %q", stderr.String())
	}
	out := stdout.String()
	first, _, _ := strings.Cut(out, "\n")
	for _, col := range []string{"SERIAL", "STATE", "KIND", "MODEL"} {
		if !strings.Contains(first, col) {
			t.Fatalf("header %q missing %s", first, col)
		}
	}
	if !strings.Contains(out, "emulator-5554") {
		t.Fatalf("serial missing from %q", out)
	}
	if !strings.Contains(out, "sdk_gphone64_x86_64") {
		t.Fatalf("model missing from %q", out)
	}
	if !strings.Contains(out, "emulator") {
		t.Fatalf("kind missing from %q", out)
	}
}

func TestDevicesJSON(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
		Model:  "sdk_gphone64_x86_64",
	}}}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"devices", "--json"},
	})
	assertDevicesJSON(t, code, stdout, stderr)
	stdout.Reset()
	stderr.Reset()
	code = run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"--json", "devices"},
	})
	assertDevicesJSON(t, code, stdout, stderr)
}

func assertDevicesJSON(t *testing.T, code int, stdout, stderr bytes.Buffer) {
	t.Helper()
	if code != ExitOK {
		t.Fatalf("exit %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr %q", stderr.String())
	}
	var got struct {
		Devices []struct {
			Serial string `json:"serial"`
			State  string `json:"state"`
			Kind   string `json:"kind"`
			Model  string `json:"model"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json: %v; stdout=%q", err, stdout.String())
	}
	if len(got.Devices) != 1 {
		t.Fatalf("len=%d, want 1; stdout=%q", len(got.Devices), stdout.String())
	}
	d := got.Devices[0]
	if d.Serial != "emulator-5554" || d.State != "device" || d.Kind != "emulator" || d.Model != "sdk_gphone64_x86_64" {
		t.Fatalf("device = %+v", d)
	}
}

func TestDevicesEmptyJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: &device.Fake{},
		args:   []string{"devices", "--json"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	var got struct {
		Devices []json.RawMessage `json:"devices"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json: %v; stdout=%q", err, stdout.String())
	}
	if got.Devices == nil {
		t.Fatalf("devices key missing or null: %q", stdout.String())
	}
	if len(got.Devices) != 0 {
		t.Fatalf("len=%d, want 0", len(got.Devices))
	}
}

func TestDevicesIncludesUnauthorized(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{Devices: []device.Device{{
		Serial: "R58Mxxx",
		State:  device.StateUnauthorized,
		Kind:   device.KindUSB,
		Model:  "SM_A715F",
	}}}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"devices"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d, want %d; stderr=%q", code, ExitOK, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "R58Mxxx") {
		t.Fatalf("serial missing from %q", out)
	}
	if !strings.Contains(out, "unauthorized") {
		t.Fatalf("state missing from %q", out)
	}
}

func TestDevicesToolMissing(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{ListErr: device.ErrToolMissing}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"devices"},
	})
	if code != ExitToolMissing {
		t.Fatalf("exit %d, want %d; stderr=%q", code, ExitToolMissing, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "tool missing") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDevicesRejectsArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: &device.Fake{},
		args:   []string{"devices", "extra"},
	})
	if code != ExitUsage {
		t.Fatalf("exit %d, want %d; stderr=%q", code, ExitUsage, stderr.String())
	}
}

func TestDevicesWriteError(t *testing.T) {
	t.Parallel()
	fake := &device.Fake{Devices: []device.Device{{
		Serial: "emulator-5554",
		State:  device.StateDevice,
		Kind:   device.KindEmulator,
	}}}
	code := run(runConfig{
		stdout: failWriter{},
		stderr: io.Discard,
		device: fake,
		args:   []string{"devices"},
	})
	if code == ExitOK {
		t.Fatalf("exit %d, want non-zero when stdout write fails", code)
	}
}
