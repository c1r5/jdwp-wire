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
	for _, p := range parsePSAll(stdout) {
		if p.Package == pkg || strings.HasPrefix(p.Package, pkg+":") {
			out = append(out, p)
		}
	}
	return out
}

func parsePSAll(stdout string) []Process {
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
		out = append(out, Process{PID: pid, Package: name})
	}
	return out
}

func parsePackageList(stdout string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		pkg, ok := strings.CutPrefix(line, "package:")
		if !ok {
			continue
		}
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		if _, dup := seen[pkg]; dup {
			continue
		}
		seen[pkg] = struct{}{}
		out = append(out, pkg)
	}
	return out
}

func parseAppLabels(stdout string) map[string]string {
	labels := map[string]string{}
	current := ""
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "Package [") {
			current = ""
			if pkg, ok := packageHeader(line); ok {
				current = pkg
			}
			continue
		}
		if current == "" {
			continue
		}
		if label, ok := labelOnLine(line); ok {
			labels[current] = label
		}
	}
	return labels
}

func packageHeader(line string) (string, bool) {
	const mark = "Package ["
	i := strings.Index(line, mark)
	if i < 0 {
		return "", false
	}
	rest := line[i+len(mark):]
	j := strings.IndexByte(rest, ']')
	if j <= 0 {
		return "", false
	}
	pkg := strings.TrimSpace(rest[:j])
	if pkg == "" || strings.ContainsAny(pkg, " \t") {
		return "", false
	}
	return pkg, true
}

func labelOnLine(line string) (string, bool) {
	const nonLoc = "nonLocalizedLabel="
	if i := strings.Index(line, nonLoc); i >= 0 {
		rest := strings.TrimSpace(line[i+len(nonLoc):])
		if cut := strings.Index(rest, " icon="); cut >= 0 {
			rest = strings.TrimSpace(rest[:cut])
		}
		if rest == "" || rest == "null" {
			return "", false
		}
		return rest, true
	}
	const appLabel = "Application Label:"
	if i := strings.Index(line, appLabel); i >= 0 {
		rest := strings.TrimSpace(line[i+len(appLabel):])
		if rest == "" || rest == "null" {
			return "", false
		}
		return rest, true
	}
	return "", false
}

func pidForPackage(pkg string, procs []Process) int {
	main, sub := 0, 0
	for _, p := range procs {
		if p.PID <= 0 {
			continue
		}
		if p.Package == pkg {
			if main == 0 || p.PID < main {
				main = p.PID
			}
			continue
		}
		if strings.HasPrefix(p.Package, pkg+":") && (sub == 0 || p.PID < sub) {
			sub = p.PID
		}
	}
	if main != 0 {
		return main
	}
	return sub
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
