package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/c1r5/jdwp-wire/internal/apps"
	"github.com/spf13/cobra"
)

const appsTimeout = 30 * time.Second

func newAppsCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "apps",
		Short: "List installed apps (PID, name, package), like frida-ps -Uai",
		Long: `List installed packages on the selected device.

IDX is 1-based. jdt pull IDX uses this same ordering. PID is blank when the app is not running.

NAME is the non-localized label when dumpsys package prints one. Otherwise NAME is the package name. Resource labels (@string/app_name) are not resolved.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(c.Context(), appsTimeout)
			defer cancel()
			serial, err := c.Flags().GetString("serial")
			if err != nil {
				return err
			}
			entries, err := apps.List(ctx, cfg.device, serial)
			if err != nil {
				return err
			}
			asJSON, err := c.Flags().GetBool("json")
			if err != nil {
				return err
			}
			if asJSON {
				return writeAppsJSON(c.OutOrStdout(), entries)
			}
			return writeAppsTable(c.OutOrStdout(), entries)
		},
	}
	c.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	return c
}

type appJSON struct {
	Index      int    `json:"index"`
	PID        *int   `json:"pid"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type appsJSON struct {
	Apps []appJSON `json:"apps"`
}

func writeAppsJSON(w io.Writer, entries []apps.Entry) error {
	out := appsJSON{Apps: make([]appJSON, 0, len(entries))}
	for _, e := range entries {
		row := appJSON{Index: e.Index, Name: e.Name, Identifier: e.Identifier}
		if e.PID > 0 {
			pid := e.PID
			row.PID = &pid
		}
		out.Apps = append(out.Apps, row)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func writeAppsTable(w io.Writer, entries []apps.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "IDX\tPID\tNAME\tIDENTIFIER"); err != nil {
		return err
	}
	for _, e := range entries {
		pid := ""
		if e.PID > 0 {
			pid = strconv.Itoa(e.PID)
		}
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", e.Index, pid, e.Name, e.Identifier); err != nil {
			return err
		}
	}
	return tw.Flush()
}
