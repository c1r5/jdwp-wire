package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/c1r5/jdwp-wire/internal/androidcli"
	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/frida"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/logging"
	"github.com/c1r5/jdwp-wire/internal/patch"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/c1r5/jdwp-wire/internal/targets"
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
const defaultJDWPTimeout = time.Minute

type runConfig struct {
	stdout      io.Writer
	stderr      io.Writer
	device      device.Client
	apk         apk.Client
	patch       patch.Applier
	jdwp        jdwp.Client
	android     androidcli.Client
	projects    projectWriter
	logger      *logging.Logger
	timeout     time.Duration
	apkTimeout  time.Duration
	jdwpTimeout time.Duration
	cwd         string
	args        []string
	frida       frida.Deps
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
	if cfg.jdwpTimeout == 0 {
		cfg.jdwpTimeout = defaultJDWPTimeout
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
	if cfg.jdwp == nil {
		cfg.jdwp = jdwp.New(execx.Exec{}, cfg.device)
	}
	if cfg.android == nil {
		cfg.android = androidcli.New(execx.Exec{})
	}
	if cfg.projects == nil {
		cfg.projects = project.FS{}
	}
	if cfg.logger == nil {
		cfg.logger = logging.New(cfg.stdout)
	}
	if cfg.cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cfg.cwd = wd
		}
	}
	if cfg.args == nil {
		cfg.args = []string{}
	}

	root := newRoot(cfg)
	root.SetArgs(cfg.args)
	root.SetOut(cfg.stdout)
	root.SetErr(cfg.stderr)

	if err := root.Execute(); err != nil {
		if _, werr := fmt.Fprintf(cfg.stderr, "jdt: %v\n", err); werr != nil {
			return 1
		}
		return exitCode(err)
	}
	return ExitOK
}

func newRoot(cfg runConfig) *cobra.Command {
	root := &cobra.Command{
		Use:           "jdt",
		Short:         "attach JDWP debug session from an Android APK/package",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       Version,
	}
	root.SetVersionTemplate("jdt {{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().Bool("json", false, "output JSON")
	root.AddCommand(newDevicesCmd(cfg))
	root.AddCommand(newAppsCmd(cfg))
	root.AddCommand(newPullCmd(cfg))
	root.AddCommand(newInstallCmd(cfg))
	root.AddCommand(newPatchCmd(cfg))
	root.AddCommand(newAttachCmd(cfg))
	root.AddCommand(newResetCmd(cfg))
	root.AddCommand(newTargetsCmd(cfg))
	root.AddCommand(newFridaSessionCmd())
	return root
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, device.ErrToolMissing), errors.Is(err, apk.ErrToolMissing), errors.Is(err, jdwp.ErrToolMissing), errors.Is(err, androidcli.ErrToolMissing), errors.Is(err, frida.ErrToolMissing):
		return ExitToolMissing
	case errors.Is(err, device.ErrNoDevice),
		errors.Is(err, device.ErrAmbiguousDevice),
		errors.Is(err, device.ErrDeviceUnusable),
		errors.Is(err, device.ErrDeviceNotFound):
		return ExitNoDevice
	case errors.Is(err, device.ErrUsage), errors.Is(err, apk.ErrUsage), errors.Is(err, patch.ErrUsage), errors.Is(err, jdwp.ErrUsage), errors.Is(err, targets.ErrUsage), errors.Is(err, project.ErrUsage), errors.Is(err, androidcli.ErrUsage), errors.Is(err, frida.ErrUsage):
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
