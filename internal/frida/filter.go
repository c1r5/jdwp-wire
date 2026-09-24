package frida

import (
	"regexp"
	"strings"
	"unicode"
)

var promptLine = regexp.MustCompile(`^\[[^\[\]\r\n]+\]->\s*$`)
var promptPrefix = regexp.MustCompile(`^\[[^\[\]\r\n]+\]->\s*`)

// Filter drops the Frida REPL banner, then rewrites each later line.
type Filter struct {
	header bool
}

// NewFilter starts in the banner. The first line that is not part of the
// banner ends it and is rewritten.
func NewFilter() *Filter {
	return &Filter{header: true}
}

// Line returns the rewritten line. keep is false when the line is dropped.
func (f *Filter) Line(raw string) (string, bool) {
	if f == nil {
		return "", false
	}
	if f.header {
		if isBanner(raw) {
			return "", false
		}
		f.header = false
	}
	out := rewrite(raw)
	if out == "" {
		return "", false
	}
	return out, true
}

func isBanner(raw string) bool {
	t := strings.TrimSpace(strings.TrimRight(raw, "\r"))
	if t == "" || bannerArt(t) {
		return true
	}
	if promptLine.MatchString(t) {
		return true
	}
	for _, n := range []string{
		"Frida ",
		"Commands:",
		"-> Displays the help",
		"object?",
		"exit/quit",
		"More info at https://frida.re",
		"Connected to",
		"Attaching",
		"Spawning",
		"Resuming",
	} {
		if strings.Contains(t, n) {
			return true
		}
	}
	return false
}

func bannerArt(t string) bool {
	for _, r := range t {
		if unicode.IsSpace(r) || strings.ContainsRune("._/\\|>-<()[]", r) {
			continue
		}
		return false
	}
	return true
}

// rewrite strips a leading Frida prompt and trailing CR. Empty results are dropped.
func rewrite(raw string) string {
	s := strings.TrimRight(raw, "\r")
	s = promptPrefix.ReplaceAllString(s, "")
	return strings.TrimRight(s, " \t")
}
