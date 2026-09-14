//go:build !unix

package execx

import "os/exec"

func configureKill(*exec.Cmd) {}
