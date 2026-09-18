package patch

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func setupDecode(t *testing.T, fixture string) string {
	t.Helper()
	dir := t.TempDir()
	b, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AndroidManifest.xml"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestApplySetsDebuggable(t *testing.T) {
	t.Parallel()
	dir := setupDecode(t, "manifest_plain.xml")
	res, err := FS{}.Apply(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Package != "com.alvo" || res.Debuggable != ActionApplied {
		t.Fatalf("%+v", res)
	}
	body, err := os.ReadFile(filepath.Join(dir, "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`android:debuggable="true"`)) {
		t.Fatalf("%s", body)
	}
	if !bytes.Contains(body, []byte(`android:label="Alvo"`)) {
		t.Fatal("rewrote whole manifest")
	}
}

func TestApplySkipsAlreadyDebuggable(t *testing.T) {
	t.Parallel()
	dir := setupDecode(t, "manifest_debuggable.xml")
	res, err := FS{}.Apply(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Debuggable != ActionSkipped {
		t.Fatalf("%+v", res)
	}
}

func TestApplyFlipsFalse(t *testing.T) {
	t.Parallel()
	dir := setupDecode(t, "manifest_false.xml")
	res, err := FS{}.Apply(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Debuggable != ActionApplied {
		t.Fatalf("%+v", res)
	}
	body, err := os.ReadFile(filepath.Join(dir, "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte(`debuggable="false"`)) {
		t.Fatalf("%s", body)
	}
}

func TestApplyNoManifest(t *testing.T) {
	t.Parallel()
	_, err := FS{}.Apply(t.TempDir())
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("%v", err)
	}
}

func TestFakeNil(t *testing.T) {
	t.Parallel()
	_, err := (*Fake)(nil).Apply("/x")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("%v", err)
	}
}
