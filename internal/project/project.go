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

type FS struct{}
