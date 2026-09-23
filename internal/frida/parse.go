package frida

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ref is one script in load order. Path is empty until a user file is checked
// or an embedded script is written to disk.
type Ref struct {
	Name string
	Path string
}

// Parse turns --bypass and repeated --script values into load order.
// Embedded names are deduped. The same user path twice is kept twice.
func Parse(bypass bool, groups []string) ([]Ref, error) {
	var out []Ref
	seen := map[string]bool{}
	addName := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		out = append(out, Ref{Name: name})
	}
	if bypass {
		for _, name := range BypassNames() {
			addName(name)
		}
	}
	for _, group := range groups {
		for _, tok := range strings.Split(group, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				return nil, fmt.Errorf("frida: script: empty: %w", ErrUsage)
			}
			if known(tok) {
				addName(tok)
				continue
			}
			if err := regularFile(tok); err != nil {
				return nil, err
			}
			out = append(out, Ref{Name: filepath.Base(tok), Path: tok})
		}
	}
	return out, nil
}

func regularFile(path string) error {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("frida: script %q: %w", path, ErrUsage)
	}
	return nil
}
