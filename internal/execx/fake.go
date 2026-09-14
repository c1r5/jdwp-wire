package execx

import "context"

type Fake struct {
	RunFn func(ctx context.Context, name string, args ...string) (Result, error)
}

func (f *Fake) Run(ctx context.Context, name string, args ...string) (Result, error) {
	if f == nil || f.RunFn == nil {
		return Result{}, ErrNotFound
	}
	return f.RunFn(ctx, name, args...)
}
