package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/logging"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/patchapply"
	"github.com/spf13/cobra"
)

func newPatchCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "patch [decoded_dir]",
		Short: "Make a decoded apktool tree debuggable and trust user CAs (destructive on the decode dir; does not install)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			apkFlag, err := c.Flags().GetString("apk")
			if err != nil {
				return err
			}
			pkgFlag, err := c.Flags().GetString("package")
			if err != nil {
				return err
			}
			asJSON, err := c.Flags().GetBool("json")
			if err != nil {
				return err
			}
			decodeDir := ""
			if len(args) == 1 {
				decodeDir = args[0]
			}
			res, err := patchapply.Run(c.Context(), patchapply.Deps{
				APK:        cfg.apk,
				Patch:      cfg.patch,
				CWD:        cfg.cwd,
				APKTimeout: cfg.apkTimeout,
			}, patchapply.Request{
				DecodeDir: decodeDir,
				APK:       apkFlag,
				Package:   pkgFlag,
			})
			if err != nil {
				return err
			}
			if abs, err := filepath.Abs(res.Dir); err == nil {
				res.Dir = abs
			}
			if asJSON {
				return writePatchJSON(c.OutOrStdout(), res)
			}
			return writePatchHuman(cfg.logger, res)
		},
	}
	c.Flags().String("apk", "", "local APK to decode then patch (requires --package)")
	c.Flags().String("package", "", "package name (required with --apk)")
	return c
}

func writePatchHuman(lg *logging.Logger, res patch.Result) error {
	if res.Debuggable == patch.ActionApplied {
		lg.OK("patch", "debuggable")
	} else {
		lg.Skip("patch", "already debuggable")
	}
	if res.NSC == patch.ActionApplied {
		lg.OK("patch", "nsc user CA")
	} else {
		lg.Skip("patch", "nsc already trusts user CA")
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
