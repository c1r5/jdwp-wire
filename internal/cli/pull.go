package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/workspace"
	"github.com/spf13/cobra"
)

func newPullCmd(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <pkg|apk>",
		Short: "Pull an installed package (base.apk) or copy a local APK into .jdt/",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.apkTimeout)
			defer cancel()
			serial, err := cmd.Flags().GetString("serial")
			if err != nil {
				return err
			}
			pkgFlag, err := cmd.Flags().GetString("package")
			if err != nil {
				return err
			}
			asJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			doDecode, err := cmd.Flags().GetBool("decode")
			if err != nil {
				return err
			}
			art, err := runPull(ctx, cfg, args[0], serial, pkgFlag)
			if err != nil {
				return err
			}
			var dec apk.Decoded
			if doDecode {
				layout, err := workspace.ForPackage(cfg.cwd, art.Package)
				if err != nil {
					return err
				}
				dec, err = cfg.apk.Decode(ctx, art.APK, layout)
				if err != nil {
					return err
				}
			}
			if asJSON {
				return writePullJSON(cmd.OutOrStdout(), art, dec.Dir)
			}
			if err := writePullHuman(cmd.OutOrStdout(), art); err != nil {
				return err
			}
			if doDecode {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "[ok] decode: %s\n", dec.Dir)
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	cmd.Flags().String("package", "", "package name (required when pulling a local APK file)")
	cmd.Flags().Bool("decode", false, "apktool-decode the APK into .jdt/<pkg>/decode")
	return cmd
}

func runPull(ctx context.Context, cfg runConfig, target, serial, pkgFlag string) (apk.Artifact, error) {
	if fileExists(target) {
		if pkgFlag == "" {
			return apk.Artifact{}, fmt.Errorf("%w: --package required for local apk", apk.ErrUsage)
		}
		layout, err := workspace.ForPackage(cfg.cwd, pkgFlag)
		if err != nil {
			return apk.Artifact{}, err
		}
		return cfg.apk.CopyAPK(ctx, target, pkgFlag, layout)
	}
	dev, err := cfg.device.Resolve(ctx, serial)
	if err != nil {
		return apk.Artifact{}, err
	}
	layout, err := workspace.ForPackage(cfg.cwd, target)
	if err != nil {
		return apk.Artifact{}, err
	}
	return cfg.apk.Pull(ctx, dev.Serial, target, layout)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func writePullHuman(w io.Writer, art apk.Artifact) error {
	if _, err := fmt.Fprintf(w, "[ok] pull: %s → %s\n", art.Package, art.APK); err != nil {
		return err
	}
	for _, s := range art.SkippedSplits {
		if _, err := fmt.Fprintf(w, "[skip] split: %s (v1)\n", s); err != nil {
			return err
		}
	}
	return nil
}

type pullJSON struct {
	Package       string   `json:"package"`
	APK           string   `json:"apk"`
	SkippedSplits []string `json:"skipped_splits"`
	Decode        string   `json:"decode,omitempty"`
}

func writePullJSON(w io.Writer, art apk.Artifact, decodeDir string) error {
	out := pullJSON{
		Package:       art.Package,
		APK:           art.APK,
		SkippedSplits: art.SkippedSplits,
		Decode:        decodeDir,
	}
	if out.SkippedSplits == nil {
		out.SkippedSplits = []string{}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
