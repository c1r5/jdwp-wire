//go:build unix

package execx

import (
	"io"
	"syscall"
	"testing"
)

func TestStartDetached_NewSession(t *testing.T) {
	t.Parallel()
	pid, stdout, err := StartDetached("sleep", "30")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_, _ = io.Copy(io.Discard, stdout)
	})
	self, err := syscall.Getpgid(0)
	if err != nil {
		t.Fatal(err)
	}
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatal(err)
	}
	if pgid == self || pgid != pid {
		t.Fatalf("pgid %d self %d pid %d", pgid, self, pid)
	}
}
