//go:build !unix

package execx

import (
	"fmt"
	"io"
)

// StartDetached is Unix-only. A detached Frida session needs a new session id.
func StartDetached(name string, _ ...string) (int, io.ReadCloser, error) {
	return 0, nil, fmt.Errorf("execx: %s: detached start is unix-only", name)
}
