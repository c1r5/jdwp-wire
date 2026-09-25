package attach

import (
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/frida"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
)

// Stop ends a session after SIGINT/SIGTERM. It kills the foreground Frida
// CLI, signals a detached frida-session, and closes the app. frida-server
// is left running. A session that never launched is left alone.
func Stop(client jdwp.Client, res Result) error {
	if res.Frida != nil {
		if res.Frida.Log != "" {
			_, _ = frida.StopSession(filepath.Dir(res.Frida.Log))
		}
		res.Frida.Stop()
	}
	if !res.Launched || client == nil {
		return nil
	}
	return client.ForceStop(res.Session.Serial, res.Session.Package, res.Session.Port)
}
