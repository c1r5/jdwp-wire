package project

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestWriteGolden(t *testing.T) {
	t.Parallel()
	l := readyLayout(t)
	got, err := FS{}.Write(Config{Layout: l, Port: 8700})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != l.Idea {
		t.Fatalf("Dir=%s", got.Dir)
	}
	if got.Decode != l.Decode {
		t.Fatalf("Decode=%s", got.Decode)
	}
	if got.IML != filepath.Join(l.Idea, "decode.iml") {
		t.Fatalf("IML=%s", got.IML)
	}
	if got.RunConfig != filepath.Join(l.Idea, ".idea", "runConfigurations", "Remote_Debug.xml") {
		t.Fatalf("RunConfig=%s", got.RunConfig)
	}
	assertFile(t, got.IML, testdata(t, "decode.iml"))
	assertFile(t, filepath.Join(l.Idea, ".idea", "misc.xml"), testdata(t, "misc.xml"))
	assertFile(t, filepath.Join(l.Idea, ".idea", "modules.xml"), testdata(t, "modules.xml"))
	assertFile(t, got.RunConfig, testdata(t, "Remote_Debug.xml"))
}

func TestWritePort(t *testing.T) {
	t.Parallel()
	l := readyLayout(t)
	got, err := FS{}.Write(Config{Layout: l, Port: 9000})
	if err != nil {
		t.Fatal(err)
	}
	body := readFile(t, got.RunConfig)
	if !strings.Contains(body, `<option name="PORT" value="9000" />`) {
		t.Fatalf("missing port 9000:\n%s", body)
	}
	if strings.Contains(body, `value="8700"`) {
		t.Fatalf("stale default port:\n%s", body)
	}
}

func TestWriteNoDecode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	l, err := workspace.ForPackage(root, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	_, err = FS{}.Write(Config{Layout: l, Port: 8700})
	if !errors.Is(err, ErrNoDecode) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriteUsage(t *testing.T) {
	t.Parallel()
	l := readyLayout(t)
	cases := []Config{
		{Layout: workspace.Layout{Idea: "", Decode: l.Decode}, Port: 8700},
		{Layout: workspace.Layout{Idea: l.Idea, Decode: ""}, Port: 8700},
		{Layout: l, Port: 0},
		{Layout: l, Port: 65536},
	}
	for _, cfg := range cases {
		_, err := FS{}.Write(cfg)
		if !errors.Is(err, ErrUsage) {
			t.Fatalf("cfg=%+v err=%v", cfg, err)
		}
	}
}

func TestWriteOverwriteKeepsStrangers(t *testing.T) {
	t.Parallel()
	l := readyLayout(t)
	if _, err := (FS{}).Write(Config{Layout: l, Port: 8700}); err != nil {
		t.Fatal(err)
	}
	stranger := filepath.Join(l.Idea, ".idea", "workspace.xml")
	if err := os.WriteFile(stranger, []byte("<project/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FS{}.Write(Config{Layout: l, Port: 9000})
	if err != nil {
		t.Fatal(err)
	}
	body := readFile(t, got.RunConfig)
	if !strings.Contains(body, `value="9000"`) {
		t.Fatalf("port not overwritten:\n%s", body)
	}
	if readFile(t, stranger) != "<project/>" {
		t.Fatal("studio file was removed")
	}
}

func readyLayout(t *testing.T) workspace.Layout {
	t.Helper()
	root := t.TempDir()
	l, err := workspace.ForPackage(root, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	man := filepath.Join(l.Decode, "AndroidManifest.xml")
	if err := os.WriteFile(man, []byte(`<manifest package="com.alvo"/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	return l
}

func testdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s\ngot:\n%s\nwant:\n%s", path, got, want)
	}
}
