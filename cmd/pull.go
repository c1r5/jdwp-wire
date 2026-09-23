package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/pull"
	"github.com/spf13/cobra"
)

func newPullCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "pull <index|pkg|apk>",
		Short: "Pull by jdt apps index, package name, or copy a local APK into .jdt/",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(c.Context(), cfg.apkTimeout)
			defer cancel()
			serial, err := c.Flags().GetString("serial")
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
			doDecode, err := c.Flags().GetBool("decode")
			if err != nil {
				return err
			}
			res, err := pull.Run(ctx, pull.Deps{
				Device: cfg.device,
				APK:    cfg.apk,
				CWD:    cfg.cwd,
			}, pull.Request{
				Target:  args[0],
				Serial:  serial,
				Package: pkgFlag,
				Decode:  doDecode,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writePullJSON(c.OutOrStdout(), res.Artifact, res.DecodeDir)
			}
			if err := writePullHuman(c.OutOrStdout(), res.Artifact); err != nil {
				return err
			}
			if doDecode {
				_, err := fmt.Fprintf(c.OutOrStdout(), "[ok] decode: %s\n", res.DecodeDir)
				return err
			}
			return nil
		},
	}
	c.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	c.Flags().String("package", "", "package name (required when pulling a local APK file)")
	c.Flags().Bool("decode", false, "apktool-decode the APK into .jdt/<pkg>/decode")
	return c
}

func writePullHuman(w io.Writer, art apk.Artifact) error {
	if _, err := fmt.Fprintf(w, "[ok] pull: %s → %s\n", art.Package, art.APK); err != nil {
		return err
	}
	for _, s := range art.Splits {
		if _, err := fmt.Fprintf(w, "[ok] pull: %s\n", s); err != nil {
			return err
		}
	}
	return nil
}

type pullJSON struct {
	Package string   `json:"package"`
	APK     string   `json:"apk"`
	Splits  []string `json:"splits"`
	Decode  string   `json:"decode,omitempty"`
}

func writePullJSON(w io.Writer, art apk.Artifact, decodeDir string) error {
	out := pullJSON{
		Package: art.Package,
		APK:     art.APK,
		Splits:  art.Splits,
		Decode:  decodeDir,
	}
	if out.Splits == nil {
		out.Splits = []string{}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
