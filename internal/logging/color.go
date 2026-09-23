package logging

import "github.com/charmbracelet/lipgloss"

// moduleColors are stable so the same module keeps the same color across runs.
var moduleColors = map[string]lipgloss.Color{
	"android":   "75",
	"attach":    "51",
	"debug-app": "135",
	"decode":    "117",
	"device":    "180",
	"encode":    "150",
	"forward":   "39",
	"install":   "120",
	"jdwp":      "45",
	"launch":    "141",
	"patch":     "213",
	"probe":     "244",
	"pull":      "81",
	"reset":     "203",
	"sign":      "228",
	"studio":    "209",
	"targets":   "86",
}

var palette = []lipgloss.Color{"213", "81", "228", "120", "141", "203", "117", "180"}

func moduleColor(name string) lipgloss.Color {
	if c, ok := moduleColors[name]; ok {
		return c
	}
	h := 0
	for _, r := range name {
		h = h*31 + int(r)
	}
	if h < 0 {
		h = -h
	}
	return palette[h%len(palette)]
}
