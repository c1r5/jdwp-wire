package apk

import (
	"os"
	"path/filepath"
	"sort"
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
			splits = append(splits, r)
			continue
		}
		others = append(others, r)
	}
	if base == "" && len(others) > 0 {
		base = others[0]
		others = others[1:]
		splits = append(splits, others...)
	} else {
		splits = append(splits, others...)
	}
	if base == "" {
		return "", splits, ErrNoBaseAPK
	}
	return base, splits, nil
}

// ListSplits returns local split_*.apk paths in dir (not base.apk).
func ListSplits(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.Contains(name, "split_") && strings.HasSuffix(name, ".apk") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out, nil
}
