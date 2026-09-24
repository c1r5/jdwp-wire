// Package frida is the optional companion that loads bypass scripts into an
// app that is already waiting for the JDWP debugger.
package frida

import "errors"

var (
	// ErrUsage is a bad script name, an empty token, or a path that is not a file.
	ErrUsage = errors.New("usage")
	// ErrToolMissing means adb or frida is not on PATH.
	ErrToolMissing = errors.New("tool missing")
)
