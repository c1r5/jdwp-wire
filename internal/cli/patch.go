package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/workspace"
	"github.com/spf13/cobra"
)

func newPatchCmd(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch [decoded_dir]",
		Short: "Make a decoded apktool tree debuggable and trust user CAs (destructive on the decode dir; does not install)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apkFlag, err := cmd.Flags().GetString("apk")
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
			res, err := runPatch(cmd.Context(), cfg, args, apkFlag, pkgFlag)
			if err != nil {
				return err
			}
			if abs, err := filepath.Abs(res.Dir); err == nil {
				res.Dir = abs
			}
			if asJSON {
				return writePatchJSON(cmd.OutOrStdout(), res)
			}
			return writePatchHuman(cmd.OutOrStdout(), res)
		},
	}
	cmd.Flags().String("apk", "", "local APK to decode then patch (requires --package)")
	cmd.Flags().String("package", "", "package name (required with --apk)")
	return cmd
}

func runPatch(ctx context.Context, cfg runConfig, args []string, apkFlag, pkgFlag string) (patch.Result, error) {
	if apkFlag != "" && len(args) > 0 {
		return patch.Result{}, fmt.Errorf("%w: decoded_dir and --apk are mutually exclusive", patch.ErrUsage)
	}
	if apkFlag != "" {
		if pkgFlag == "" {
			return patch.Result{}, fmt.Errorf("%w: --package required for --apk", patch.ErrUsage)
		}
		if !fileExists(apkFlag) {
			return patch.Result{}, fmt.Errorf("patch: apk: %w", os.ErrNotExist)
		}
		layout, err := workspace.ForPackage(cfg.cwd, pkgFlag)
		if err != nil {
			return patch.Result{}, err
		}
		ctx, cancel := context.WithTimeout(ctx, cfg.apkTimeout)
		defer cancel()
		dec, err := cfg.apk.Decode(ctx, apkFlag, layout)
		if err != nil {
			return patch.Result{}, err
		}
		return cfg.patch.Apply(dec.Dir)
	}
	if len(args) != 1 {
		return patch.Result{}, fmt.Errorf("%w: decoded_dir or --apk required", patch.ErrUsage)
	}
	st, err := os.Stat(args[0])
	if err != nil {
		return patch.Result{}, err
	}
	if !st.IsDir() {
		return patch.Result{}, fmt.Errorf("%w: decoded_dir must be a directory", patch.ErrUsage)
	}
	return cfg.patch.Apply(args[0])
}

func writePatchHuman(w io.Writer, res patch.Result) error {
	if res.Debuggable == patch.ActionApplied {
		if _, err := fmt.Fprintln(w, "[ok] patch: debuggable"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "[skip] patch: already debuggable"); err != nil {
			return err
		}
	}
	if res.NSC == patch.ActionApplied {
		if _, err := fmt.Fprintln(w, "[ok] patch: nsc user CA"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "[skip] patch: nsc already trusts user CA"); err != nil {
			return err
		}
	}
	return nil
}

type patchJSON struct {
	Package    string `json:"package"`
	Dir        string `json:"dir"`
	Debuggable string `json:"debuggable"`
	NSC        string `json:"nsc"`
}

func writePatchJSON(w io.Writer, res patch.Result) error {
	b, err := json.Marshal(patchJSON{
		Package:    res.Package,
		Dir:        res.Dir,
		Debuggable: string(res.Debuggable),
		NSC:        string(res.NSC),
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
