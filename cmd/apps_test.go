package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestAppsHumanAndJSON(t *testing.T) {
	t.Parallel()
	fake := testDevice()
	fake.Apps = []device.App{
		{Package: "com.zeta", Label: "Zeta", PID: 111},
		{Package: "com.alpha", Label: "Alpha"},
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"apps"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	first, _, _ := strings.Cut(out, "\n")
	for _, col := range []string{"IDX", "PID", "NAME", "IDENTIFIER"} {
		if !strings.Contains(first, col) {
			t.Fatalf("header %q missing %s", first, col)
		}
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines %q", out)
	}
	if !strings.Contains(lines[1], "111") || !strings.Contains(lines[1], "com.zeta") {
		t.Fatalf("running app %q", lines[1])
	}
	if !strings.Contains(lines[2], "Alpha") || !strings.Contains(lines[2], "com.alpha") {
		t.Fatalf("stopped app %q", lines[2])
	}
	if strings.Contains(lines[2], "111") {
		t.Fatalf("stopped app has pid %q", lines[2])
	}

	stdout.Reset()
	stderr.Reset()
	code = run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"apps", "--json"},
	})
	if code != ExitOK {
		t.Fatalf("json exit %d stderr=%q", code, stderr.String())
	}
	var body struct {
		Apps []struct {
			Index      int    `json:"index"`
			PID        *int   `json:"pid"`
			Name       string `json:"name"`
			Identifier string `json:"identifier"`
		} `json:"apps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Apps) != 2 || body.Apps[0].Identifier != "com.zeta" || body.Apps[0].PID == nil || *body.Apps[0].PID != 111 {
		t.Fatalf("%+v", body.Apps)
	}
	if body.Apps[1].Identifier != "com.alpha" || body.Apps[1].PID != nil {
		t.Fatalf("%+v", body.Apps[1])
	}
}

func TestAppsHidesSystemUnlessFlag(t *testing.T) {
	t.Parallel()
	fake := testDevice()
	fake.Apps = []device.App{
		{Package: "com.user", Label: "User"},
		{Package: "com.android.settings", Label: "Settings", System: true},
	}
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"apps"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "settings") {
		t.Fatalf("default listed system app %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "com.user") {
		t.Fatalf("missing user app %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: fake,
		args:   []string{"apps", "--system"},
	})
	if code != ExitOK {
		t.Fatalf("system exit %d stderr=%q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 || !strings.Contains(lines[1], "com.android.settings") || !strings.Contains(lines[2], "com.user") {
		t.Fatalf("lines %q", stdout.String())
	}
}

func TestAppsNoDevice(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: &device.Fake{},
		args:   []string{"apps"},
	})
	if code != ExitNoDevice {
		t.Fatalf("exit %d want %d stderr=%q", code, ExitNoDevice, stderr.String())
	}
}

func TestAppsEmptyJSON(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run(runConfig{
		stdout: &stdout,
		stderr: &stderr,
		device: testDevice(),
		args:   []string{"--json", "apps"},
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"apps":[]}` {
		t.Fatalf("stdout %s", stdout.String())
	}
}
