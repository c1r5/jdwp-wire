//go:build unix

package frida

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func killPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err == nil || err == syscall.ESRCH {
		return nil
	}
	return err
}

// StopSession signals the detached frida-session pid, if one was recorded.
// A missing file or a dead pid is not an error. stopped is true when a live pid was signaled.
func StopSession(dir string) (bool, error) {
	path := pidPath(dir)
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("frida: session: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	_ = os.Remove(path)
	if err != nil || pid <= 0 {
		return false, nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			return false, nil
		}
		return false, fmt.Errorf("frida: session: %w", err)
	}
	return true, nil
}
