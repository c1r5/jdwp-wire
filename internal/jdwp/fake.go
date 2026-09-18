package jdwp

import (
	"context"
	"fmt"
)

type Fake struct {
	AttachFn func(ctx context.Context, serial, pkg string, port int) (Session, error)
	ResetFn  func(ctx context.Context, serial, pkg string, port int) error
}

func fakeErr() error {
	return fmt.Errorf("jdwp: fake: %w", ErrUsage)
}

func (f *Fake) Attach(ctx context.Context, serial, pkg string, port int) (Session, error) {
	if f == nil || f.AttachFn == nil {
		return Session{}, fakeErr()
	}
	return f.AttachFn(ctx, serial, pkg, port)
}

func (f *Fake) Reset(ctx context.Context, serial, pkg string, port int) error {
	if f == nil || f.ResetFn == nil {
		return fakeErr()
	}
	return f.ResetFn(ctx, serial, pkg, port)
}

var _ Client = (*Fake)(nil)
