package project

import (
	"errors"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

func TestFakeWrite(t *testing.T) {
	t.Parallel()
	f := &Fake{
		WriteFn: func(cfg Config) (Result, error) {
			if cfg.Port != 8700 {
				t.Fatalf("port %d", cfg.Port)
			}
			return Result{Dir: cfg.Layout.Idea, RunConfig: "Remote_Debug.xml"}, nil
		},
	}
	got, err := f.Write(Config{Layout: workspace.Layout{Idea: "/x/idea", Decode: "/x/decode"}, Port: 8700})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != "/x/idea" {
		t.Fatalf("Dir=%s", got.Dir)
	}
}

func TestFakeNil(t *testing.T) {
	t.Parallel()
	var f *Fake
	_, err := f.Write(Config{})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}
