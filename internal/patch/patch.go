package patch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrUsage         = errors.New("usage")
	ErrNoManifest    = errors.New("no AndroidManifest.xml")
	ErrNoApplication = errors.New("no application element")
)

type Action string

const (
	ActionApplied Action = "applied"
	ActionSkipped Action = "skipped"
)

type Result struct {
	Package    string
	Dir        string
	Debuggable Action
	NSC        Action
}

type Applier interface {
	Apply(dir string) (Result, error)
}

type FS struct{}

var packageAttr = regexp.MustCompile(`package="([^"]+)"`)

func (FS) Apply(dir string) (Result, error) {
	if dir == "" {
		return Result{}, fmt.Errorf("patch: apply: %w", ErrUsage)
	}
	manPath := filepath.Join(dir, "AndroidManifest.xml")
	body, err := os.ReadFile(manPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf("patch: apply: %w", ErrNoManifest)
		}
		return Result{}, fmt.Errorf("patch: apply: %w", err)
	}
	pkg := "unknown"
	if m := packageAttr.FindSubmatch(body); len(m) == 2 {
		pkg = string(m[1])
	}
	start, end, err := applicationSpan(body)
	if err != nil {
		return Result{}, fmt.Errorf("patch: apply: %w", err)
	}
	tag, deb := setDebuggable(body[start:end])
	if deb == ActionApplied {
		out := append(append([]byte{}, body[:start]...), append(tag, body[end:]...)...)
		if err := os.WriteFile(manPath, out, 0o644); err != nil {
			return Result{}, fmt.Errorf("patch: apply: %w", err)
		}
	}
	return Result{
		Package:    pkg,
		Dir:        dir,
		Debuggable: deb,
		NSC:        ActionSkipped,
	}, nil
}
