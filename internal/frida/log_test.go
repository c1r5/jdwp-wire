package frida

import (
	"os"
	"testing"
	"time"
)

func TestLog_AppendsDailyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Date(2026, 9, 23, 15, 4, 5, 0, time.UTC)
	log, err := OpenLog(dir, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.WriteLine("sslpinning-bypass armed"); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	log, err = OpenLog(dir, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.WriteLine("antiroot-bypass armed"); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(log.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "sslpinning-bypass armed\nantiroot-bypass armed\n" {
		t.Fatalf("body %q", body)
	}
	if LogPath(dir, now) != log.Path() {
		t.Fatalf("path %s", log.Path())
	}
}
