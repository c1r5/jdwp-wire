package apk

import (
	"path/filepath"
	"strings"
)

func parsePMPath(stdout string) (base string, splits []string, err error) {
	var remotes []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path, ok := strings.CutPrefix(line, "package:")
		if !ok {
			continue
		}
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		remotes = append(remotes, path)
	}
	if len(remotes) == 0 {
		return "", nil, ErrPackageNotFound
	}
	var others []string
	for _, r := range remotes {
		name := filepath.Base(r)
		if name == "base.apk" {
			base = r
			continue
		}
		if strings.Contains(name, "split_") {
			splits = append(splits, name)
			continue
		}
		others = append(others, r)
	}
	if base == "" && len(others) > 0 {
		base = others[0]
		others = others[1:]
		for _, r := range others {
			splits = append(splits, filepath.Base(r))
		}
	}
	if base == "" {
		return "", splits, ErrNoBaseAPK
	}
	return base, splits, nil
}
