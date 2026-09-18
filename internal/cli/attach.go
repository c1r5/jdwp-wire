package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/spf13/cobra"
)

func newAttachCmd(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach <pkg>",
		Short: "Forward JDWP for an already-installed debuggable app (does not patch or install)",
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
			sess, err := cfg.jdwp.Attach(ctx, dev.Serial, args[0], port)
			if err != nil {
				return err
			}
			if asJSON {
				return writeAttachJSON(cmd.OutOrStdout(), sess)
			}
			return writeAttachHuman(cmd.OutOrStdout(), sess)
		},
	}
	cmd.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	cmd.Flags().Int("port", 8700, "local TCP port to forward")
	cmd.Flags().Bool("studio", false, "print Studio attach hint (project generation is not wired)")
	return cmd
}

func writeAttachHuman(w io.Writer, sess jdwp.Session) error {
	if _, err := fmt.Fprintf(w, "[ok] debug-app: %s\n", sess.Package); err != nil {
		return err
	}
	if sess.Activity != "" {
		if _, err := fmt.Fprintf(w, "[ok] launch: %s\n", sess.Activity); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "[ok] launch: monkey %s\n", sess.Package); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "[ok] jdwp: pid %d\n", sess.PID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "[ok] forward: tcp:%d -> jdwp:%d\n", sess.Port, sess.PID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "[ok] probe: 127.0.0.1:%d\n", sess.Port); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[skip] patch: use jdt patch and jdt install first"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[skip] studio: not wired"); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "attach: localhost:%d\n", sess.Port)
	return err
}

type attachJSON struct {
	Package  string `json:"package"`
	Serial   string `json:"serial"`
	PID      int    `json:"pid"`
	Port     int    `json:"port"`
	Activity string `json:"activity"`
}

func writeAttachJSON(w io.Writer, sess jdwp.Session) error {
	b, err := json.Marshal(attachJSON{
		Package:  sess.Package,
		Serial:   sess.Serial,
		PID:      sess.PID,
		Port:     sess.Port,
		Activity: sess.Activity,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
