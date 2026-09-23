//go:build unix

package execx

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

// StartDetached starts name in a new session. The parent exiting does not
// kill it. stdout is the child's stdout and stderr together. The caller kills
// the returned pid, which is the session leader.
func StartDetached(name string, args ...string) (int, io.ReadCloser, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, fmt.Errorf("execx: %s: %w", name, err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			return 0, nil, fmt.Errorf("execx: %s: %w", name, fmt.Errorf("%w: %w", ErrNotFound, exec.ErrNotFound))
		}
		return 0, nil, fmt.Errorf("execx: %s: %w", name, err)
	}
	return cmd.Process.Pid, stdout, nil
}
