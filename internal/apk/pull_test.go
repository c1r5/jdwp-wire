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
	if len(splits) != 1 || !strings.Contains(splits[0], "split_config.xxhdpi.apk") {
		t.Fatalf("splits=%v", splits)
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
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		if name != "adb" {
			t.Fatalf("name %s", name)
		}
		if args[0] == "-s" && args[2] == "shell" {
			return execx.Result{Stdout: "package:/data/app/x/base.apk\npackage:/data/app/x/split_config.xxhdpi.apk\n"}, nil
		}
		if args[0] == "-s" && args[2] == "pull" {
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
	if len(art.SkippedSplits) != 1 {
		t.Fatalf("skipped %v", art.SkippedSplits)
	}
	b, err := os.ReadFile(art.APK)
	if err != nil || string(b) != "apk" {
		t.Fatalf("file %q err %v", b, err)
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
