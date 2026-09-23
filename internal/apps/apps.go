package apps

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/device"
)

// Entry is one row of jdt apps. Index is 1-based in display order.
type Entry struct {
	Index      int
	PID        int
	Name       string
	Identifier string
}

// List returns installed apps in the same order jdt apps prints.
// includeSystem adds system packages. Otherwise only third-party packages are returned.
// Running apps (PID > 0) come first; each group is ordered by name, then package.
func List(ctx context.Context, c device.Client, serial string, includeSystem bool) ([]Entry, error) {
	dev, err := c.Resolve(ctx, serial)
	if err != nil {
		return nil, err
	}
	raw, err := c.ListApps(ctx, dev.Serial, includeSystem)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(raw))
	for _, app := range raw {
		if app.Package == "" {
			continue
		}
		pid := app.PID
		if pid < 0 {
			pid = 0
		}
		entries = append(entries, Entry{
			PID:        pid,
			Name:       displayName(app),
			Identifier: app.Package,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		ri := entries[i].PID > 0
		rj := entries[j].PID > 0
		if ri != rj {
			return ri
		}
		ni := strings.ToLower(entries[i].Name)
		nj := strings.ToLower(entries[j].Name)
		if ni != nj {
			return ni < nj
		}
		return entries[i].Identifier < entries[j].Identifier
	})
	for i := range entries {
		entries[i].Index = i + 1
	}
	return entries, nil
}

// PackageAt returns the package at a 1-based index from List.
func PackageAt(entries []Entry, index int) (string, error) {
	if index < 1 || index > len(entries) {
		return "", fmt.Errorf("%w: app index %d out of range (1-%d)", device.ErrUsage, index, len(entries))
	}
	return entries[index-1].Identifier, nil
}

func displayName(app device.App) string {
	if name := strings.TrimSpace(app.Label); name != "" {
		return name
	}
	return app.Package
}
