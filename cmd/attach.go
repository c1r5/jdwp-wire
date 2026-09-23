package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/c1r5/jdwp-wire/internal/attach"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/spf13/cobra"
)

type projectWriter interface {
	Write(cfg project.Config) (project.Result, error)
}

func newAttachCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "attach <pkg>",
		Short: "Forward JDWP for an already-installed debuggable app (does not patch or install)",
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
			studio, err := c.Flags().GetBool("studio")
			if err != nil {
				return err
			}
			asJSON, err := c.Flags().GetBool("json")
			if err != nil {
				return err
			}
			res, err := attach.Run(ctx, cfg.device, cfg.jdwp, attach.Options{
				Serial:  serial,
				Package: args[0],
				Port:    port,
				CWD:     cfg.cwd,
				Studio:  studio,
				Writer:  cfg.projects,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writeAttachJSON(c.OutOrStdout(), res)
			}
			return writeAttachHuman(c.OutOrStdout(), res)
		},
	}
	c.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	c.Flags().Int("port", 8700, "local TCP port to forward")
	c.Flags().Bool("studio", false, "write .jdt/<pkg>/idea for Android Studio (does not launch the IDE)")
	return c
}

func writeAttachHuman(w io.Writer, res attach.Result) error {
	sess := res.Session
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
	var studio string
	switch res.Studio {
	case attach.StudioWritten:
		studio = fmt.Sprintf("[ok] studio: %s", res.Project.Dir)
	case attach.StudioNoDecode:
		studio = "[skip] studio: no decode"
	default:
		studio = "[skip] studio: pass --studio"
	}
	if _, err := fmt.Fprintln(w, studio); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "attach: localhost:%d\n", sess.Port)
	return err
}

type studioJSON struct {
	Dir     string `json:"dir"`
	Skipped bool   `json:"skipped"`
	Reason  string `json:"reason,omitempty"`
}

type attachJSON struct {
	Package  string      `json:"package"`
	Serial   string      `json:"serial"`
	PID      int         `json:"pid"`
	Port     int         `json:"port"`
	Activity string      `json:"activity"`
	Studio   *studioJSON `json:"studio,omitempty"`
}

func writeAttachJSON(w io.Writer, res attach.Result) error {
	sess := res.Session
	out := attachJSON{
		Package:  sess.Package,
		Serial:   sess.Serial,
		PID:      sess.PID,
		Port:     sess.Port,
		Activity: sess.Activity,
	}
	switch res.Studio {
	case attach.StudioWritten:
		out.Studio = &studioJSON{Dir: res.Project.Dir}
	case attach.StudioNoDecode:
		out.Studio = &studioJSON{Skipped: true, Reason: "no decode"}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
