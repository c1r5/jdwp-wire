package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newInstallCmd(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <apk>",
		Short: "Install an APK with adb install -r -d",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.apkTimeout)
			defer cancel()
			serial, err := cmd.Flags().GetString("serial")
			if err != nil {
				return err
			}
			asJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			apkPath := args[0]
			if _, err := os.Stat(apkPath); err != nil {
				return err
			}
			dev, err := cfg.device.Resolve(ctx, serial)
			if err != nil {
				return err
			}
			if err := cfg.apk.Install(ctx, dev.Serial, apkPath); err != nil {
				return err
			}
			if asJSON {
				return writeInstallJSON(cmd.OutOrStdout(), apkPath, dev.Serial)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "[ok] install: %s\n", apkPath)
			return err
		},
	}
	cmd.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	return cmd
}

type installJSON struct {
	APK    string `json:"apk"`
	Serial string `json:"serial"`
}

func writeInstallJSON(w io.Writer, apkPath, serial string) error {
	b, err := json.Marshal(installJSON{APK: apkPath, Serial: serial})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
