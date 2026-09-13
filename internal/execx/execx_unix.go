//go:build unix

package execx

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

// waitDelay bounds cmd.Run after Cancel when a descendant still holds the pipes.
const waitDelay = 100 * time.Millisecond

func configureKill(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == nil || errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	cmd.WaitDelay = waitDelay
}
