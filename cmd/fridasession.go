package cmd

import (
	"os/signal"
	"syscall"

	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/frida"
	"github.com/spf13/cobra"
)

func newFridaSessionCmd() *cobra.Command {
	var serial, pkg, logPath string
	var pid, port int
	var scripts []string
	c := &cobra.Command{
		Use:    "frida-session",
		Hidden: true,
		Short:  "internal Frida log supervisor",
		Args:   cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(c.Context(), syscall.SIGTERM, syscall.SIGINT)
			defer stop()
			return frida.Serve(ctx, frida.Deps{Run: execx.Exec{}}, serial, pkg, pid, port, logPath, scripts, c.OutOrStdout())
		},
	}
	c.Flags().StringVar(&serial, "serial", "", "adb serial")
	c.Flags().StringVar(&pkg, "package", "", "package to close on SIGTERM")
	c.Flags().IntVar(&pid, "pid", 0, "app pid")
	c.Flags().IntVar(&port, "port", 0, "forwarded JDWP port")
	c.Flags().StringVar(&logPath, "log", "", "daily log path")
	c.Flags().StringArrayVar(&scripts, "script", nil, "script path")
	return c
}
