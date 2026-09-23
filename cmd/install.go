package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/c1r5/jdwp-wire/internal/install"
	"github.com/c1r5/jdwp-wire/internal/logging"
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
			res, err := install.Run(ctx, install.Deps{
				Device: cfg.device,
				APK:    cfg.apk,
				CWD:    cfg.cwd,
			}, install.Request{
				Target:  args[0],
				Serial:  serial,
				Package: pkgFlag,
			})
			if err != nil {
				return err
			}
			apks := append([]string{res.APK}, res.Splits...)
			if asJSON {
				return writeInstallJSON(c.OutOrStdout(), res.APK, res.Splits, res.Serial)
			}
			return writeInstallHuman(cfg.logger, apks, res.Encoded)
		},
	}
	c.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	c.Flags().String("package", "", "package name (required when installing a decode dir outside .jdt/<pkg>/decode)")
	return c
}

func writeInstallHuman(lg *logging.Logger, apks []string, encoded bool) error {
	if len(apks) == 0 {
		return nil
	}
	if encoded {
		lg.OK("encode", apks[0])
	}
	for _, p := range apks {
		lg.OK("sign", p)
	}
	for _, p := range apks {
		lg.OK("install", p)
	}
	return nil
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
