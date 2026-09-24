//go:build !unix

package frida

import "fmt"

func killPID(int) error { return fmt.Errorf("frida: session: unix-only") }

// StopSession is Unix-only.
func StopSession(string) (bool, error) {
	return false, fmt.Errorf("frida: session: unix-only")
}
