package frida

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed scripts/*.js
var embedded embed.FS

// Materialize writes embedded scripts into dir and leaves user paths as they are.
// The returned refs are in load order and each Path is a file frida can -l.
func Materialize(dir string, refs []Ref) ([]Ref, error) {
	if len(refs) == 0 {
		return nil, fmt.Errorf("frida: script: empty: %w", ErrUsage)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("frida: script: %w", err)
	}
	out := make([]Ref, len(refs))
	for i, ref := range refs {
		if ref.Path != "" {
			out[i] = ref
			continue
		}
		body, err := embedded.ReadFile("scripts/" + ref.Name + ".js")
		if err != nil {
			return nil, fmt.Errorf("frida: script %q: %w", ref.Name, ErrUsage)
		}
		dst := filepath.Join(dir, ref.Name+".js")
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			return nil, fmt.Errorf("frida: script: %w", err)
		}
		out[i] = Ref{Name: ref.Name, Path: dst}
	}
	return out, nil
}
