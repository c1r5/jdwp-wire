package frida

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.js")
	if err := os.WriteFile(custom, []byte("console.log(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	again := filepath.Join(dir, "custom.js")

	got, err := Parse(true, []string{"sslpinning-bypass," + custom, again})
	if err != nil {
		t.Fatal(err)
	}
	want := []Ref{
		{Name: NameRoot},
		{Name: NameDebug},
		{Name: NameSSL},
		{Name: "custom.js", Path: custom},
		{Name: "custom.js", Path: again},
	}
	if len(got) != len(want) {
		t.Fatalf("len %d %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		bypass bool
		groups []string
	}{
		{false, []string{""}},
		{false, []string{"nope"}},
		{false, []string{"a,,b"}},
		{false, []string{filepath.Join(t.TempDir(), "missing.js")}},
		{false, []string{t.TempDir()}},
	}
	for _, tc := range cases {
		_, err := Parse(tc.bypass, tc.groups)
		if !errors.Is(err, ErrUsage) {
			t.Fatalf("Parse(%v %v) err = %v", tc.bypass, tc.groups, err)
		}
	}
}

func TestParse_ScriptOnly(t *testing.T) {
	t.Parallel()
	got, err := Parse(false, []string{NameSSL})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != NameSSL || got[0].Path != "" {
		t.Fatalf("%+v", got)
	}
}
