package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/workspace"
	"github.com/spf13/cobra"
)

// Consumer-side contracts for --pull / --decode / --patch.
// Commands depend on these, not on concrete apk/patch types.
// apk.Client and patch.Applier satisfy them; other fakes can too.

type resolver interface {
	Resolve(ctx context.Context, serial string) (device.Device, error)
}

type puller interface {
	Pull(ctx context.Context, serial, pkg string, layout workspace.Layout) (stepAPK, error)
}

type decoder interface {
	Decode(ctx context.Context, apkPath string, layout workspace.Layout) (stepDecode, error)
}

type stepAPK struct {
	Package string
	Path    string
	Skipped []string
}

type stepDecode struct {
	Package string
	Dir     string
}

type stepState struct {
	Layout workspace.Layout
	APK    stepAPK
	Decode stepDecode
}

func addStepFlags(cmd *cobra.Command, pull, decode, patchFlag bool) {
	if pull {
		cmd.Flags().Bool("pull", false, "pull the installed package into .jdt/<pkg>/apk")
	}
	if decode {
		cmd.Flags().Bool("decode", false, "apktool-decode the APK into .jdt/<pkg>/decode")
	}
	if patchFlag {
		cmd.Flags().Bool("patch", false, "patch the decoded tree (debuggable + NSC user CA)")
	}
}

func flagBool(cmd *cobra.Command, name string) (bool, error) {
	if cmd.Flags().Lookup(name) == nil {
		return false, nil
	}
	return cmd.Flags().GetBool(name)
}

type pipeline struct {
	resolve resolver
	pull    puller
	decode  decoder
	cwd     string
}

func pipelineFrom(cfg runConfig) pipeline {
	return pipeline{
		resolve: cfg.device,
		pull:    apkPullAdapter{cfg.apk},
		decode:  apkDecodeAdapter{cfg.apk},
		cwd:     cfg.cwd,
	}
}

func (p pipeline) run(ctx context.Context, pkg, serial string, doPull, doDecode bool) (stepState, error) {
	layout, err := workspace.ForPackage(p.cwd, pkg)
	if err != nil {
		return stepState{}, err
	}
	st := stepState{Layout: layout}
	if doPull {
		if p.pull == nil || p.resolve == nil {
			return stepState{}, fmt.Errorf("pipeline: pull: %w", apk.ErrUsage)
		}
		dev, err := p.resolve.Resolve(ctx, serial)
		if err != nil {
			return stepState{}, err
		}
		art, err := p.pull.Pull(ctx, dev.Serial, pkg, layout)
		if err != nil {
			return stepState{}, err
		}
		st.APK = art
	}
	if doDecode {
		if p.decode == nil {
			return stepState{}, fmt.Errorf("pipeline: decode: %w", apk.ErrUsage)
		}
		apkPath := st.APK.Path
		if apkPath == "" {
			apkPath = filepath.Join(layout.APK, "base.apk")
		}
		dec, err := p.decode.Decode(ctx, apkPath, layout)
		if err != nil {
			return stepState{}, err
		}
		st.Decode = dec
	}
	return st, nil
}

func writeStepHuman(w io.Writer, doPull, doDecode bool, st stepState) error {
	if doPull {
		if err := writePullHuman(w, apkArtifact(st.APK)); err != nil {
			return err
		}
	}
	if doDecode {
		if _, err := fmt.Fprintf(w, "[ok] decode: %s\n", st.Decode.Dir); err != nil {
			return err
		}
	}
	return nil
}

type apkPullAdapter struct{ c apk.Client }

func (a apkPullAdapter) Pull(ctx context.Context, serial, pkg string, layout workspace.Layout) (stepAPK, error) {
	if a.c == nil {
		return stepAPK{}, apk.ErrUsage
	}
	art, err := a.c.Pull(ctx, serial, pkg, layout)
	if err != nil {
		return stepAPK{}, err
	}
	return stepAPK{Package: art.Package, Path: art.APK, Skipped: art.SkippedSplits}, nil
}

type apkDecodeAdapter struct{ c apk.Client }

func (a apkDecodeAdapter) Decode(ctx context.Context, apkPath string, layout workspace.Layout) (stepDecode, error) {
	if a.c == nil {
		return stepDecode{}, apk.ErrUsage
	}
	d, err := a.c.Decode(ctx, apkPath, layout)
	if err != nil {
		return stepDecode{}, err
	}
	return stepDecode{Package: d.Package, Dir: d.Dir}, nil
}

func apkArtifact(s stepAPK) apk.Artifact {
	return apk.Artifact{Package: s.Package, APK: s.Path, SkippedSplits: s.Skipped}
}
