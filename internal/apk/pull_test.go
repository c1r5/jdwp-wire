package apk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestParsePMPathBaseAndSplits(t *testing.T) {
	t.Parallel()
	in := "package:/data/app/~~x==/com.alvo-x/base.apk\npackage:/data/app/~~x==/com.alvo-x/split_config.xxhdpi.apk\n"
	base, splits, err := parsePMPath(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(base, "/base.apk") {
		t.Fatalf("base=%s", base)
	}
	wantSplit := "/data/app/~~x==/com.alvo-x/split_config.xxhdpi.apk"
	if len(splits) != 1 || splits[0] != wantSplit {
		t.Fatalf("splits=%v, want %q", splits, wantSplit)
	}
}

func TestParsePMPathEmpty(t *testing.T) {
	t.Parallel()
	_, _, err := parsePMPath("")
	if !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestParsePMPathSplitsOnly(t *testing.T) {
	t.Parallel()
	_, _, err := parsePMPath("package:/data/app/x/split_config.en.apk\n")
	if !errors.Is(err, ErrNoBaseAPK) {
		t.Fatalf("err=%v", err)
	}
}

func TestToolsPull(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	layout, err := workspace.ForPackage(dir, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	var pulled []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		if name != "adb" {
			t.Fatalf("name %s", name)
		}
		if args[0] == "-s" && args[2] == "shell" {
			return execx.Result{Stdout: "package:/data/app/x/base.apk\npackage:/data/app/x/split_config.xxhdpi.apk\n"}, nil
		}
		if args[0] == "-s" && args[2] == "pull" {
			pulled = append(pulled, args[3])
			dest := args[4]
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return execx.Result{}, err
			}
			return execx.Result{}, os.WriteFile(dest, []byte("apk"), 0o644)
		}
		return execx.Result{}, errors.New("unexpected " + strings.Join(args, " "))
	}}
	art, err := New(r).Pull(context.Background(), "emu", "com.alvo", layout)
	if err != nil {
		t.Fatal(err)
	}
	if art.Package != "com.alvo" {
		t.Fatalf("pkg %s", art.Package)
	}
	if filepath.Base(art.APK) != "base.apk" {
		t.Fatalf("apk %s", art.APK)
	}
	if len(art.Splits) != 1 || filepath.Base(art.Splits[0]) != "split_config.xxhdpi.apk" {
		t.Fatalf("splits %v", art.Splits)
	}
	wantRemote := []string{"/data/app/x/base.apk", "/data/app/x/split_config.xxhdpi.apk"}
	if len(pulled) != 2 || pulled[0] != wantRemote[0] || pulled[1] != wantRemote[1] {
		t.Fatalf("pulled %v", pulled)
	}
	for _, p := range append([]string{art.APK}, art.Splits...) {
		b, err := os.ReadFile(p)
		if err != nil || string(b) != "apk" {
			t.Fatalf("file %s %q err %v", p, b, err)
		}
	}
}

func TestListSplits(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "base.apk"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	split := filepath.Join(dir, "split_config.xxhdpi.apk")
	if err := os.WriteFile(split, []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ListSplits(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != split {
		t.Fatalf("got %v", got)
	}
}

func TestCopyAPK(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "in.apk")
	if err := os.WriteFile(src, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	layout, err := workspace.ForPackage(dir, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	art, err := New(&execx.Fake{}).CopyAPK(context.Background(), src, "com.alvo", layout)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(art.APK)
	if err != nil || string(b) != "apk" {
		t.Fatalf("copied %q err %v", b, err)
	}
	if art.Package != "com.alvo" {
		t.Fatalf("pkg %s", art.Package)
	}
}
