package apk

import (
	"context"
	"fmt"

	"github.com/c1r5/jdwp-wire/internal/workspace"
)

type Fake struct {
	PullFn    func(ctx context.Context, serial, pkg string, layout workspace.Layout) (Artifact, error)
	CopyFn    func(ctx context.Context, src, pkg string, layout workspace.Layout) (Artifact, error)
	DecodeFn  func(ctx context.Context, apkPath string, layout workspace.Layout) (Decoded, error)
	BuildFn   func(ctx context.Context, decodedDir, outAPK string) (Artifact, error)
	SignFn    func(ctx context.Context, apkPath, keystore string) (Artifact, error)
	InstallFn func(ctx context.Context, serial string, apks ...string) error
}

func fakeErr() error {
	return fmt.Errorf("apk: fake: %w", ErrUsage)
}

func (f *Fake) Pull(ctx context.Context, serial, pkg string, layout workspace.Layout) (Artifact, error) {
	if f == nil || f.PullFn == nil {
		return Artifact{}, fakeErr()
	}
	return f.PullFn(ctx, serial, pkg, layout)
}

func (f *Fake) CopyAPK(ctx context.Context, src, pkg string, layout workspace.Layout) (Artifact, error) {
	if f == nil || f.CopyFn == nil {
		return Artifact{}, fakeErr()
	}
	return f.CopyFn(ctx, src, pkg, layout)
}

func (f *Fake) Decode(ctx context.Context, apkPath string, layout workspace.Layout) (Decoded, error) {
	if f == nil || f.DecodeFn == nil {
		return Decoded{}, fakeErr()
	}
	return f.DecodeFn(ctx, apkPath, layout)
}

func (f *Fake) Build(ctx context.Context, decodedDir, outAPK string) (Artifact, error) {
	if f == nil || f.BuildFn == nil {
		return Artifact{}, fakeErr()
	}
	return f.BuildFn(ctx, decodedDir, outAPK)
}

func (f *Fake) Sign(ctx context.Context, apkPath, keystore string) (Artifact, error) {
	if f == nil || f.SignFn == nil {
		return Artifact{}, fakeErr()
	}
	return f.SignFn(ctx, apkPath, keystore)
}

func (f *Fake) Install(ctx context.Context, serial string, apks ...string) error {
	if f == nil || f.InstallFn == nil {
		return fakeErr()
	}
	return f.InstallFn(ctx, serial, apks...)
}
