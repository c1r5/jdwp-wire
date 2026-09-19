package jdwp

import (
	"strconv"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func parseActivity(stdout string) string {
	var last string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.ContainsAny(line, " \t") {
			continue
		}
		if !strings.Contains(line, "/") {
			continue
		}
		last = line
	}
	return last
}

func parseJDWP(stdout string) []int {
	var out []int
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n, err := strconv.Atoi(line)
		if err != nil || n <= 0 {
			continue
		}
		out = append(out, n)
	}
	return out
}

func pickPID(pkg string, procs []device.Process) int {
	for _, p := range procs {
		if p.Package == pkg && p.PID > 0 {
			return p.PID
		}
	}
	return 0
}
