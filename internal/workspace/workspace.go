package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Layout struct {
	Root     string
	APK      string
	Decode   string
	Patched  string
	Idea     string
	Captures string
}

func ForPackage(cwd, pkg string) (Layout, error) {
	if err := validatePackage(pkg); err != nil {
		return Layout{}, fmt.Errorf("workspace: %w", err)
	}
	root := filepath.Join(cwd, ".jdt", pkg)
	return Layout{
		Root:     root,
		APK:      filepath.Join(root, "apk"),
		Decode:   filepath.Join(root, "decode"),
		Patched:  filepath.Join(root, "patched"),
		Idea:     filepath.Join(root, "idea"),
		Captures: filepath.Join(root, "captures"),
	}, nil
}

func (l Layout) Ensure() error {
	for _, p := range []string{l.Root, l.APK, l.Decode, l.Patched, l.Idea, l.Captures} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("workspace: ensure: %w", err)
		}
	}
	return nil
}

func DebugKeystore(cwd string) string {
	return filepath.Join(cwd, ".jdt", "debug.keystore")
}

func validatePackage(pkg string) error {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return fmt.Errorf("empty package")
	}
	if strings.Contains(pkg, "..") {
		return fmt.Errorf("invalid package %q", pkg)
	}
	if pkg != filepath.Base(pkg) {
		return fmt.Errorf("invalid package %q", pkg)
	}
	if strings.ContainsAny(pkg, `/\`) {
		return fmt.Errorf("invalid package %q", pkg)
	}
	return nil
}
