package workspace

import (
	"os"
	"testing"
)

func TestForPackage(t *testing.T) {
	t.Parallel()
	l, err := ForPackage("/tmp/proj", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if l.Root != "/tmp/proj/.jdt/com.alvo" {
		t.Fatalf("Root=%s", l.Root)
	}
	if l.APK != "/tmp/proj/.jdt/com.alvo/apk" {
		t.Fatalf("APK=%s", l.APK)
	}
	if l.Decode != "/tmp/proj/.jdt/com.alvo/decode" {
		t.Fatalf("Decode=%s", l.Decode)
	}
	if l.Patched != "/tmp/proj/.jdt/com.alvo/patched" {
		t.Fatalf("Patched=%s", l.Patched)
	}
	if l.Idea != "/tmp/proj/.jdt/com.alvo/idea" {
		t.Fatalf("Idea=%s", l.Idea)
	}
	if l.Captures != "/tmp/proj/.jdt/com.alvo/captures" {
		t.Fatalf("Captures=%s", l.Captures)
	}
}

func TestForPackageRejects(t *testing.T) {
	t.Parallel()
	for _, pkg := range []string{"", "com/alvo", `com\alvo`, "..", "com/../evil"} {
		if _, err := ForPackage("/tmp/proj", pkg); err == nil {
			t.Fatalf("accepted %q", pkg)
		}
	}
}

func TestEnsureIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l, err := ForPackage(dir, "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{l.Root, l.APK, l.Decode, l.Patched, l.Idea, l.Captures} {
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			t.Fatalf("%s: %v", p, err)
		}
	}
}

func TestDebugKeystore(t *testing.T) {
	t.Parallel()
	if got := DebugKeystore("/tmp/proj"); got != "/tmp/proj/.jdt/debug.keystore" {
		t.Fatalf("got %s", got)
	}
}
