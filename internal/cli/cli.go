package cli

import (
	"fmt"
	"io"
	"os"
)

// Version is the CLI version string printed by --version.
const Version = "0.0.0"

const (
	ExitOK          = 0
	ExitUsage       = 2
	ExitNoDevice    = 3
	ExitToolMissing = 4
)

const usage = `jdt — attach JDWP debug session from an Android APK/package

Usage:
  jdt --help
  jdt --version

Commands are not wired yet. See AGENTS.md for the target CLI.
`

// Run executes the CLI with process stdio.
func Run(args []string) int {
	return RunWith(os.Stdout, os.Stderr, args)
}

// RunWith executes the CLI against the given writers (tests).
func RunWith(stdout, stderr io.Writer, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return ExitOK
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return ExitOK
	case "-v", "--version", "version":
		fmt.Fprintln(stdout, "jdt", Version)
		return ExitOK
	default:
		fmt.Fprintf(stderr, "jdt: unknown command %q\n", args[0])
		return ExitUsage
	}
}
