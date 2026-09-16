package apk

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestDecodeInvokesApktool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	layout, err := workspace.ForPackage(dir, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	apkPath := filepath.Join(layout.APK, "base.apk")
	var got []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		got = append([]string{name}, args...)
		return execx.Result{}, nil
	}}
	d, err := New(r).Decode(context.Background(), apkPath, layout)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"apktool", "d", "-f", "-o", layout.Decode, apkPath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if d.Dir != layout.Decode || d.Package != "com.alvo" {
		t.Fatalf("%+v", d)
	}
}

func TestDecodeToolMissing(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{}, execx.ErrNotFound
	}}
	_, err := New(r).Decode(context.Background(), "/x.apk", workspace.Layout{Root: "/p/.jdt/com.alvo", Decode: "/p/.jdt/com.alvo/decode"})
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err=%v", err)
	}
}

func TestBuildInvokesApktool(t *testing.T) {
	t.Parallel()
	var got []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		got = append([]string{name}, args...)
		return execx.Result{}, nil
	}}
	art, err := New(r).Build(context.Background(), "/decode", "/out.apk")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"apktool", "b", "-o", "/out.apk", "/decode"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if art.APK != "/out.apk" {
		t.Fatalf("apk %s", art.APK)
	}
}

func TestDecodeExitTruncatesStderr(t *testing.T) {
	t.Parallel()
	long := make([]byte, 3000)
	for i := range long {
		long[i] = 'x'
	}
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{Stderr: string(long)}, &execx.ExitError{Name: "apktool", ExitCode: 1, Stderr: string(long)}
	}}
	_, err := New(r).Decode(context.Background(), "/x.apk", workspace.Layout{Root: "/p/.jdt/com.alvo", Decode: "/p/.jdt/com.alvo/decode"})
	if !errors.Is(err, execx.ErrExit) {
		t.Fatalf("err=%v", err)
	}
	if len(err.Error()) > 2500 {
		t.Fatalf("stderr not truncated: %d", len(err.Error()))
	}
}
