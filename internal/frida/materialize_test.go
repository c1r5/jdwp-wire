package frida

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialize_WritesEmbedded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.js")
	if err := os.WriteFile(custom, []byte("console.log('x')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err := Parse(true, []string{custom})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Materialize(filepath.Join(dir, "frida"), refs)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 || out[3].Path != custom {
		t.Fatalf("%+v", out)
	}
	for _, ref := range out[:3] {
		body, err := os.ReadFile(ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if !strings.Contains(text, "Java.perform") {
			t.Fatalf("%s missing Java.perform", ref.Name)
		}
		for _, banned := range []string{"Play Integrity", "SafetyNet", "DroidGuard", "FLAG_DEBUGGABLE", "waitForDebugger"} {
			if strings.Contains(text, banned) {
				t.Fatalf("%s contains %s", ref.Name, banned)
			}
		}
	}
}
