package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/patch"
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
const defaultAPKTimeout = 5 * time.Minute

type runConfig struct {
	stdout     io.Writer
	stderr     io.Writer
	device     device.Client
	apk        apk.Client
	patch      patch.Applier
	timeout    time.Duration
	apkTimeout time.Duration
	cwd        string
	args       []string
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
	if cfg.apkTimeout == 0 {
		cfg.apkTimeout = defaultAPKTimeout
	}
	if cfg.device == nil {
		cfg.device = device.NewADB(execx.Exec{})
	}
	if cfg.apk == nil {
		cfg.apk = apk.New(execx.Exec{})
	}
	if cfg.patch == nil {
		cfg.patch = patch.FS{}
	}
	if cfg.cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cfg.cwd = wd
		}
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
	cmd.AddCommand(newPullCmd(cfg))
	cmd.AddCommand(newInstallCmd(cfg))
	cmd.AddCommand(newPatchCmd(cfg))
	return cmd
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, device.ErrToolMissing), errors.Is(err, apk.ErrToolMissing):
		return ExitToolMissing
	case errors.Is(err, device.ErrNoDevice),
		errors.Is(err, device.ErrAmbiguousDevice),
		errors.Is(err, device.ErrDeviceUnusable),
		errors.Is(err, device.ErrDeviceNotFound):
		return ExitNoDevice
	case errors.Is(err, device.ErrUsage), errors.Is(err, apk.ErrUsage), errors.Is(err, patch.ErrUsage):
		return ExitUsage
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "flag needs an argument") ||
		strings.Contains(msg, "accepts") && strings.Contains(msg, "arg(s)") {
		return ExitUsage
	}
	return 1
}
