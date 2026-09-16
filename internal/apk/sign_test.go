package apk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestSignUsesApksigner(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "a.apk")
	ks := filepath.Join(dir, "debug.keystore")
	if err := os.WriteFile(apkPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var names []string
	tools := New(&execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		names = append(names, name)
		if name == "keytool" {
			return execx.Result{}, os.WriteFile(ks, []byte("ks"), 0o644)
		}
		return execx.Result{}, nil
	}})
	tools.look = func(n string) (string, error) {
		if n == "apksigner" {
			return "/bin/apksigner", nil
		}
		return "", execx.ErrNotFound
	}
	if _, err := tools.Sign(context.Background(), apkPath, ks); err != nil {
		t.Fatal(err)
	}
	if len(names) < 2 || names[0] != "keytool" || names[1] != "apksigner" {
		t.Fatalf("names %v", names)
	}
}

func TestSignUsesUberWithoutKeytool(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "a.apk")
	if err := os.WriteFile(apkPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var names []string
	tools := New(&execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		names = append(names, name)
		return execx.Result{}, nil
	}})
	tools.look = func(n string) (string, error) {
		if n == "uber-apk-signer" {
			return "/bin/uber-apk-signer", nil
		}
		return "", execx.ErrNotFound
	}
	if _, err := tools.Sign(context.Background(), apkPath, filepath.Join(dir, "debug.keystore")); err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "uber-apk-signer" {
		t.Fatalf("names %v", names)
	}
}

func TestSignToolMissing(t *testing.T) {
	t.Parallel()
	tools := New(&execx.Fake{})
	tools.look = func(string) (string, error) { return "", execx.ErrNotFound }
	_, err := tools.Sign(context.Background(), "/a.apk", "/ks")
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err=%v", err)
	}
}
