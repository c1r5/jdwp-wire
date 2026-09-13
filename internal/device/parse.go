package device

import (
	"strings"
)

func parseDevices(stdout string) []Device {
	var out []Device
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "List of devices attached" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		serial := fields[0]
		if serial == "*" || strings.HasPrefix(serial, "*") {
			continue
		}
		out = append(out, Device{
			Serial: serial,
			State:  parseState(fields[1]),
			Kind:   parseKind(serial, fields),
			Model:  parseModel(fields),
		})
	}
	return out
}

func parseState(s string) State {
	switch s {
	case string(StateDevice):
		return StateDevice
	case string(StateOffline):
		return StateOffline
	case string(StateUnauthorized):
		return StateUnauthorized
	default:
		return StateUnknown
	}
}

func parseKind(serial string, fields []string) Kind {
	if strings.HasPrefix(serial, "emulator-") {
		return KindEmulator
	}
	for _, f := range fields {
		if strings.HasPrefix(f, "usb:") {
			return KindUSB
		}
	}
	return KindUnknown
}

func parseModel(fields []string) string {
	for _, f := range fields {
		if strings.HasPrefix(f, "model:") {
			return strings.TrimPrefix(f, "model:")
		}
	}
	return ""
}
