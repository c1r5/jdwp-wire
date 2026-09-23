package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/muesli/termenv"
)

func fixedClock(l *Logger) {
	l.raw.SetTimeFunction(func(time.Time) time.Time {
		return time.Date(2026, 9, 23, 15, 4, 5, 0, time.UTC)
	})
}

func TestStepPlain(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	fixedClock(l)
	l.OK("patch", "debuggable")
	l.Skip("patch", "already debuggable")
	got := buf.String()
	if strings.Contains(got, "\x1b") {
		t.Fatalf("buffer should stay plain, got %q", got)
	}
	want := "15:04:05 [ok] [patch] debuggable\n15:04:05 [skip] [patch] already debuggable\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestModuleColorsDiffer(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	l.raw.SetColorProfile(termenv.TrueColor)
	fixedClock(l)
	l.OK("patch", "debuggable")
	patch := buf.String()
	buf.Reset()
	l.OK("pull", "base.apk")
	pull := buf.String()
	if !strings.Contains(patch, "\x1b") || !strings.Contains(pull, "\x1b") {
		t.Fatalf("expected color\npatch %q\npull %q", patch, pull)
	}
	if !strings.Contains(patch, "[ok]") || !strings.Contains(patch, "[patch]") {
		t.Fatalf("patch line %q", patch)
	}
	if !strings.Contains(pull, "[pull]") {
		t.Fatalf("pull line %q", pull)
	}
	if patch == pull {
		t.Fatal("patch and pull rendered the same")
	}
}

func TestUnknownModule(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	fixedClock(l)
	l.OK("custom", "hello")
	if buf.String() != "15:04:05 [ok] [custom] hello\n" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestNilWriter(t *testing.T) {
	l := New(nil)
	l.OK("patch", "debuggable")
}
