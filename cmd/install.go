package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/c1r5/jdwp-wire/internal/install"
	"github.com/spf13/cobra"
)

func newInstallCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "install <apk|decoded_dir|pkg>",
		Short: "Rebuild a decoded tree if needed, sign debug, and adb install -r -d",
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
			lg := cfg.logger
			if asJSON {
				lg = nil
			}
			res, err := install.Run(ctx, install.Deps{
				Device: cfg.device,
				APK:    cfg.apk,
				CWD:    cfg.cwd,
				Log:    lg,
			}, install.Request{
				Target:  args[0],
				Serial:  serial,
				Package: pkgFlag,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writeInstallJSON(c.OutOrStdout(), res.APK, res.Splits, res.Serial)
			}
			return nil
		},
	}
	c.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	c.Flags().String("package", "", "package name (required when installing a decode dir outside .jdt/<pkg>/decode)")
	return c
}

type installJSON struct {
	APK    string   `json:"apk"`
	Splits []string `json:"splits"`
	Serial string   `json:"serial"`
}

func writeInstallJSON(w io.Writer, apkPath string, splits []string, serial string) error {
	if splits == nil {
		splits = []string{}
	}
	b, err := json.Marshal(installJSON{APK: apkPath, Splits: splits, Serial: serial})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
