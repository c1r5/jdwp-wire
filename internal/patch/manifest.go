package patch

import (
	"bytes"
	"regexp"
)

var (
	applicationOpen = regexp.MustCompile(`(?s)<application\b[^>]*?/?>`)
	debuggableTrue  = regexp.MustCompile(`android:debuggable\s*=\s*['"]true['"]`)
	debuggableFalse = regexp.MustCompile(`android:debuggable\s*=\s*['"]false['"]`)
)

func applicationSpan(body []byte) (start, end int, err error) {
	loc := applicationOpen.FindIndex(body)
	if loc == nil {
		return 0, 0, ErrNoApplication
	}
	return loc[0], loc[1], nil
}

func setDebuggable(tag []byte) ([]byte, Action) {
	if debuggableTrue.Match(tag) {
		return tag, ActionSkipped
	}
	if debuggableFalse.Match(tag) {
		out := debuggableFalse.ReplaceAll(tag, []byte(`android:debuggable="true"`))
		return out, ActionApplied
	}
	return insertAttr(tag, ` android:debuggable="true"`), ActionApplied
}

func insertAttr(tag []byte, attr string) []byte {
	if bytes.HasSuffix(tag, []byte("/>")) {
		return append(append([]byte{}, tag[:len(tag)-2]...), append([]byte(attr), '/', '>')...)
	}
	if len(tag) > 0 && tag[len(tag)-1] == '>' {
		return append(append([]byte{}, tag[:len(tag)-1]...), append([]byte(attr), '>')...)
	}
	return tag
}
