package project

import "fmt"

type Fake struct {
	WriteFn func(cfg Config) (Result, error)
}

func (f *Fake) Write(cfg Config) (Result, error) {
	if f == nil || f.WriteFn == nil {
		return Result{}, fmt.Errorf("project: fake: %w", ErrUsage)
	}
	return f.WriteFn(cfg)
}
