package frida

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteLoaderDefersJavaUntilAttach(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "custom.js")
	body := "Java.perform(function () {\n  console.log('sg.vantagepoint.a.b');\n});\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := writeLoader(dir, []Ref{{Name: "custom.js", Path: src}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{
		"Application.attach",
		"currentApplication",
		"Java hooks wait for Application.attach",
		`console.log('sg.vantagepoint.a.b')`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "handleBindApplication") {
		t.Fatal("loader hooks the method that is already on the stack")
	}
}
