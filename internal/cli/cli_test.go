package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{nil, {"--help"}, {"-h"}, {"help"}} {
		var stdout, stderr bytes.Buffer
		code := RunWith(&stdout, &stderr, args)
		if code != ExitOK {
			t.Fatalf("args %v: exit %d, want %d", args, code, ExitOK)
		}
		if !strings.Contains(stdout.String(), "jdt") {
			t.Fatalf("args %v: help missing from stdout: %q", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("args %v: unexpected stderr %q", args, stderr.String())
		}
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := RunWith(&stdout, &stderr, []string{"--version"})
	if code != ExitOK {
		t.Fatalf("exit %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), Version) {
		t.Fatalf("version missing from stdout: %q", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := RunWith(&stdout, &stderr, []string{"attach"})
	if code != ExitUsage {
		t.Fatalf("exit %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), `unknown command "attach"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout %q", stdout.String())
	}
}
