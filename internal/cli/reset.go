package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newResetCmd(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset <pkg>",
		Short: "Clear set-debug-app and remove the JDWP forward",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.jdwpTimeout)
			defer cancel()
			serial, err := cmd.Flags().GetString("serial")
			if err != nil {
				return err
			}
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				return err
			}
			asJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			dev, err := cfg.device.Resolve(ctx, serial)
			if err != nil {
				return err
			}
			pkg := args[0]
			if err := cfg.jdwp.Reset(ctx, dev.Serial, pkg, port); err != nil {
				return err
			}
			if asJSON {
				return writeResetJSON(cmd.OutOrStdout(), pkg, dev.Serial, port)
			}
			return writeResetHuman(cmd.OutOrStdout(), pkg, port)
		},
	}
	cmd.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	cmd.Flags().Int("port", 8700, "local TCP port whose forward should be removed")
	return cmd
}

func writeResetHuman(w io.Writer, pkg string, port int) error {
	if _, err := fmt.Fprintf(w, "[ok] reset: clear-debug-app %s\n", pkg); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "[ok] reset: remove forward tcp:%d\n", port)
	return err
}

type resetJSON struct {
	Package string `json:"package"`
	Serial  string `json:"serial"`
	Port    int    `json:"port"`
}

func writeResetJSON(w io.Writer, pkg, serial string, port int) error {
	b, err := json.Marshal(resetJSON{Package: pkg, Serial: serial, Port: port})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
