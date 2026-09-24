package frida

const (
	// NameRoot hides common root and Magisk checks.
	NameRoot = "antiroot-bypass"
	// NameDebug hides debugger-connected and TracerPid checks.
	NameDebug = "antidebug-bypass"
	// NameSSL disables common Java TLS pinning.
	NameSSL = "sslpinning-bypass"
)

// BypassNames is the --bypass set, root and debug first so they arm before TLS.
func BypassNames() []string {
	return []string{NameRoot, NameDebug, NameSSL}
}

func known(name string) bool {
	switch name {
	case NameRoot, NameDebug, NameSSL:
		return true
	default:
		return false
	}
}
