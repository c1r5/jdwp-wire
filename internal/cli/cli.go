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
		return writeOK(stdout, usage)
	}
	switch args[0] {
	case "-h", "--help", "help":
		return writeOK(stdout, usage)
	case "-v", "--version", "version":
		if _, err := fmt.Fprintln(stdout, "jdt", Version); err != nil {
			return 1
		}
		return ExitOK
	default:
		if _, err := fmt.Fprintf(stderr, "jdt: unknown command %q\n", args[0]); err != nil {
			return 1
		}
		return ExitUsage
	}
}

func writeOK(w io.Writer, s string) int {
	if _, err := io.WriteString(w, s); err != nil {
		return 1
	}
	return ExitOK
}
