package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/spf13/cobra"
)

func newDevicesCmd(cfg runConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List connected Android devices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.timeout)
			defer cancel()
			devs, err := cfg.device.List(ctx)
			if err != nil {
				return err
			}
			asJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			if asJSON {
				return writeDevicesJSON(cmd.OutOrStdout(), devs)
			}
			return writeDevicesTable(cmd.OutOrStdout(), devs)
		},
	}
}

type deviceJSON struct {
	Serial string `json:"serial"`
	State  string `json:"state"`
	Kind   string `json:"kind"`
	Model  string `json:"model"`
}

type devicesJSON struct {
	Devices []deviceJSON `json:"devices"`
}

func writeDevicesJSON(w io.Writer, devs []device.Device) error {
	out := devicesJSON{Devices: make([]deviceJSON, 0, len(devs))}
	for _, d := range devs {
		out.Devices = append(out.Devices, deviceJSON{
			Serial: d.Serial,
			State:  string(d.State),
			Kind:   string(d.Kind),
			Model:  d.Model,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func writeDevicesTable(w io.Writer, devs []device.Device) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SERIAL\tSTATE\tKIND\tMODEL"); err != nil {
		return err
	}
	for _, d := range devs {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", d.Serial, d.State, d.Kind, d.Model); err != nil {
			return err
		}
	}
	return tw.Flush()
}
