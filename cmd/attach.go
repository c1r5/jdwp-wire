package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/attach"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/spf13/cobra"
)

type projectWriter interface {
	Write(cfg project.Config) (project.Result, error)
}

func newAttachCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "attach <pkg>",
		Short: "Decode, patch, install, and forward JDWP for an installed package",
		Long: `Attach a JDWP session for an installed package.

The pipeline always starts with pull and decode and ends with install, then adb forward on the device that is still connected. A step that is already done is skipped on its own: an existing apktool tree skips decode, and a manifest that is already debuggable or already trusts user CAs skips that part of the patch. Later steps still run. If that serial is gone after install, forward does not run.

Install replaces the installed app with a debug build signed by jdt. A signature mismatch uninstalls the package first, which clears its data. When the official Android CLI is on PATH, install and launch use android run --debug. Otherwise adb install and am set-debug-app are used.

--studio writes .jdt/<pkg>/idea after the forward and does not open Android Studio.`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(c.Context(), cfg.apkTimeout+cfg.jdwpTimeout)
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
				APK:     cfg.apk,
				Patch:   cfg.patch,
				Android: cfg.android,
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

func formatDevice(d device.Device) string {
	parts := []string{string(d.Kind), d.Serial}
	if d.Model != "" {
		parts = append(parts, d.Model)
	}
	return strings.Join(parts, " ")
}

func writeAttachHuman(w io.Writer, res attach.Result) error {
	if res.SkipDecode {
		if _, err := fmt.Fprintln(w, "[skip] pull: decode already exists"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "[skip] decode: %s\n", res.DecodeDir); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "[ok] pull: %s\n", res.PullAPK); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "[ok] decode: %s\n", res.DecodeDir); err != nil {
			return err
		}
	}
	if err := writePatchHuman(w, res.Patch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "[ok] sign: %s\n", res.Signed); err != nil {
		return err
	}
	if res.Launch == attach.LaunchAndroid {
		if _, err := fmt.Fprintln(w, "[ok] launch: android run --debug"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "[skip] android: cli unavailable"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "[ok] install: %s\n", res.Signed); err != nil {
			return err
		}
	}
	sess := res.Session
	if res.Launch != attach.LaunchAndroid {
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
	}
	if _, err := fmt.Fprintf(w, "[ok] jdwp: pid %d\n", sess.PID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "[ok] device: %s\n", formatDevice(res.Device)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "[ok] forward: adb -s %s tcp:%d -> jdwp:%d\n", sess.Serial, sess.Port, sess.PID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "[ok] probe: 127.0.0.1:%d\n", sess.Port); err != nil {
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
	Package    string      `json:"package"`
	Serial     string      `json:"serial"`
	PID        int         `json:"pid"`
	Port       int         `json:"port"`
	Activity   string      `json:"activity"`
	Repackaged bool        `json:"repackaged"`
	Launch     string      `json:"launch"`
	Studio     *studioJSON `json:"studio,omitempty"`
}

func writeAttachJSON(w io.Writer, res attach.Result) error {
	sess := res.Session
	out := attachJSON{
		Package:    sess.Package,
		Serial:     sess.Serial,
		PID:        sess.PID,
		Port:       sess.Port,
		Activity:   sess.Activity,
		Repackaged: res.Repackaged,
		Launch:     string(res.Launch),
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
