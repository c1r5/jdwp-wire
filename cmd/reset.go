package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/frida"
	"github.com/c1r5/jdwp-wire/internal/logging"
	"github.com/c1r5/jdwp-wire/internal/reset"
	"github.com/c1r5/jdwp-wire/internal/workspace"
	"github.com/spf13/cobra"
)

func newResetCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "reset <pkg>",
		Short: "Clear set-debug-app and remove the JDWP forward",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(c.Context(), cfg.jdwpTimeout)
			defer cancel()
			serial, err := c.Flags().GetString("serial")
			if err != nil {
				return err
			}
			port, err := c.Flags().GetInt("port")
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
			armed := false
			if s, ok := cfg.jdwp.(interface{ SetLog(*logging.Logger) }); ok {
				s.SetLog(lg)
				armed = lg != nil
			}
			res, err := reset.Run(ctx, cfg.device, cfg.jdwp, serial, args[0], port)
			if err != nil {
				return err
			}
			layout, err := workspace.ForPackage(cfg.cwd, args[0])
			if err != nil {
				return err
			}
			stopped, err := frida.StopSession(filepath.Join(layout.Root, "frida"))
			if err != nil {
				return err
			}
			if stopped {
				lg.OK("frida", "session stopped")
			}
			if asJSON {
				return writeResetJSON(c.OutOrStdout(), res.Package, res.Serial, res.Port)
			}
			if armed {
				return nil
			}
			return writeResetHuman(cfg.logger, res.Package, res.Port)
		},
	}
	c.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	c.Flags().Int("port", 8700, "local TCP port whose forward should be removed")
	return c
}

func writeResetHuman(lg *logging.Logger, pkg string, port int) error {
	lg.OK("reset", "clear-debug-app "+pkg)
	lg.OK("reset", fmt.Sprintf("remove forward tcp:%d", port))
	return nil
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
