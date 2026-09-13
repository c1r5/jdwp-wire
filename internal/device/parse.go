package device

import (
	"sort"
	"strconv"
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

func parsePidof(stdout string) []int {
	var pids []int
	for _, f := range strings.Fields(stdout) {
		n, err := strconv.Atoi(f)
		if err != nil || n <= 0 {
			continue
		}
		pids = append(pids, n)
	}
	return pids
}

func parsePS(stdout, pkg string) []Process {
	var out []Process
	pidCol, nameCol := -1, -1
	headerSeen := false
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if !headerSeen {
			for i, f := range fields {
				switch strings.ToUpper(f) {
				case "PID":
					pidCol = i
				case "NAME", "COMMAND", "CMD", "ARGS":
					nameCol = i
				}
			}
			if pidCol >= 0 {
				headerSeen = true
				continue
			}
		}
		pid, name := pickPIDName(fields, pidCol, nameCol)
		if pid <= 0 || name == "" {
			continue
		}
		if name == pkg || strings.HasPrefix(name, pkg+":") {
			out = append(out, Process{PID: pid, Package: name})
		}
	}
	return out
}

func pickPIDName(fields []string, pidCol, nameCol int) (int, string) {
	if pidCol >= 0 && pidCol < len(fields) && nameCol >= 0 && nameCol < len(fields) {
		n, err := strconv.Atoi(fields[pidCol])
		if err != nil {
			return 0, ""
		}
		return n, fields[len(fields)-1]
	}
	if len(fields) < 2 {
		return 0, ""
	}
	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err == nil && n > 0 {
			return n, fields[len(fields)-1]
		}
	}
	return 0, ""
}

func mergeProcesses(pkg string, pids []int, fromPS []Process) []Process {
	byPID := make(map[int]Process, len(pids)+len(fromPS))
	for _, pid := range pids {
		byPID[pid] = Process{PID: pid, Package: pkg}
	}
	for _, p := range fromPS {
		byPID[p.PID] = p
	}
	var defaults, rest []Process
	for _, p := range byPID {
		if p.Package == pkg {
			defaults = append(defaults, p)
		} else {
			rest = append(rest, p)
		}
	}
	sort.Slice(defaults, func(i, j int) bool { return defaults[i].PID < defaults[j].PID })
	sort.Slice(rest, func(i, j int) bool { return rest[i].PID < rest[j].PID })
	return append(defaults, rest...)
}
