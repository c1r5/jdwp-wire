package project

import (
	"errors"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

var (
	ErrUsage    = errors.New("usage")
	ErrNoDecode = errors.New("no decode")
)

type Config struct {
	Layout workspace.Layout
	Port   int
}

type Result struct {
	Dir       string
	Decode    string
	IML       string
	RunConfig string
}

// Writer writes a minimal IntelliJ project under layout.Idea.
// Temporary: move to the consumer when cli attach --studio lands.
type Writer interface {
	Write(cfg Config) (Result, error)
}

type FS struct{}
