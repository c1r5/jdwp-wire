package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/spf13/cobra"
)

// Version is the CLI version string printed by --version.
const Version = "0.0.0"

const (
	ExitOK          = 0
	ExitUsage       = 2
	ExitNoDevice    = 3
	ExitToolMissing = 4
)

const defaultTimeout = 10 * time.Second

type runConfig struct {
	stdout  io.Writer
	stderr  io.Writer
	device  device.Client
	timeout time.Duration
	args    []string
}

// Run executes the CLI with process stdio.
func Run(args []string) int {
	return RunWith(os.Stdout, os.Stderr, args)
}

// RunWith executes the CLI against the given writers (tests).
func RunWith(stdout, stderr io.Writer, args []string) int {
	return run(runConfig{stdout: stdout, stderr: stderr, args: args})
}

func run(cfg runConfig) int {
	if cfg.stdout == nil {
		cfg.stdout = os.Stdout
	}
	if cfg.stderr == nil {
		cfg.stderr = os.Stderr
	}
	if cfg.timeout == 0 {
		cfg.timeout = defaultTimeout
	}
	if cfg.device == nil {
		cfg.device = device.NewADB(execx.Exec{})
	}
	if cfg.args == nil {
		cfg.args = []string{}
	}

	cmd := newRoot(cfg)
	cmd.SetArgs(cfg.args)
	cmd.SetOut(cfg.stdout)
	cmd.SetErr(cfg.stderr)

	if err := cmd.Execute(); err != nil {
		if _, werr := fmt.Fprintf(cfg.stderr, "jdt: %v\n", err); werr != nil {
			return 1
		}
		return exitCode(err)
	}
	return ExitOK
}

func newRoot(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "jdt",
		Short:         "attach JDWP debug session from an Android APK/package",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       Version,
	}
	cmd.SetVersionTemplate("jdt {{.Version}}\n")
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.PersistentFlags().Bool("json", false, "output JSON")
	cmd.AddCommand(newDevicesCmd(cfg))
	return cmd
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, device.ErrToolMissing):
		return ExitToolMissing
	case errors.Is(err, device.ErrNoDevice):
		return ExitNoDevice
	case errors.Is(err, device.ErrUsage):
		return ExitUsage
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "flag needs an argument") {
		return ExitUsage
	}
	return 1
}
