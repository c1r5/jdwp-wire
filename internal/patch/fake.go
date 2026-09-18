package patch

import "fmt"

type Fake struct {
	ApplyFn func(dir string) (Result, error)
}

func (f *Fake) Apply(dir string) (Result, error) {
	if f == nil || f.ApplyFn == nil {
		return Result{}, fmt.Errorf("patch: fake: %w", ErrUsage)
	}
	return f.ApplyFn(dir)
}
