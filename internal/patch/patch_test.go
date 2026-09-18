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

func TestApplyWritesDefaultNSC(t *testing.T) {
	t.Parallel()
	dir := setupDecode(t, "manifest_plain.xml")
	res, err := FS{}.Apply(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.NSC != ActionApplied {
		t.Fatalf("%+v", res)
	}
	body, err := os.ReadFile(filepath.Join(dir, "res", "xml", "network_security_config.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`src="user"`)) {
		t.Fatalf("%s", body)
	}
	man, err := os.ReadFile(filepath.Join(dir, "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(man, []byte(`android:networkSecurityConfig="@xml/network_security_config"`)) {
		t.Fatalf("%s", man)
	}
}

func TestApplyMergesExistingNSC(t *testing.T) {
	t.Parallel()
	dir := setupDecode(t, "manifest_plain.xml")
	man := []byte(`<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="com.alvo">
    <application android:label="Alvo" android:networkSecurityConfig="@xml/nsc">
    </application>
</manifest>`)
	if err := os.WriteFile(filepath.Join(dir, "AndroidManifest.xml"), man, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "res", "xml"), 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("testdata/nsc_system.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "res", "xml", "nsc.xml"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := FS{}.Apply(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.NSC != ActionApplied {
		t.Fatalf("%+v", res)
	}
	got, err := os.ReadFile(filepath.Join(dir, "res", "xml", "nsc.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte(`src="user"`)) {
		t.Fatalf("%s", got)
	}
}

func TestApplySkipsUserNSC(t *testing.T) {
	t.Parallel()
	dir := setupDecode(t, "manifest_debuggable.xml")
	if err := os.MkdirAll(filepath.Join(dir, "res", "xml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "res", "xml", "network_security_config.xml"),
		[]byte(`<network-security-config><trust-anchors><certificates src="user" /></trust-anchors></network-security-config>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AndroidManifest.xml"), []byte(`<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="com.alvo">
    <application android:debuggable="true" android:networkSecurityConfig="@xml/network_security_config">
    </application>
</manifest>`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := FS{}.Apply(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.NSC != ActionSkipped || res.Debuggable != ActionSkipped {
		t.Fatalf("%+v", res)
	}
}
