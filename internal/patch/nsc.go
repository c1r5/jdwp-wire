package patch

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const defaultNSC = `<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <base-config cleartextTrafficPermitted="true">
        <trust-anchors>
            <certificates src="system" />
            <certificates src="user" />
        </trust-anchors>
    </base-config>
</network-security-config>
`

const baseConfigBlock = `    <base-config cleartextTrafficPermitted="true">
        <trust-anchors>
            <certificates src="system" />
            <certificates src="user" />
        </trust-anchors>
    </base-config>
`

var (
	nscAttr     = regexp.MustCompile(`android:networkSecurityConfig\s*=\s*['"]@xml/([A-Za-z0-9_.]+)['"]`)
	userCert    = regexp.MustCompile(`src\s*=\s*['"]user['"]`)
	trustClose  = []byte("</trust-anchors>")
	nscOpen     = regexp.MustCompile(`<network-security-config[^>]*>`)
	userCertXML = []byte("            <certificates src=\"user\" />\n")
)

func setNSCAttr(tag []byte) (out []byte, name string, inserted bool) {
	if m := nscAttr.FindSubmatch(tag); len(m) == 2 {
		return tag, string(m[1]), false
	}
	return insertAttr(tag, ` android:networkSecurityConfig="@xml/network_security_config"`), "network_security_config", true
}

func ensureNSC(path string) (Action, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(defaultNSC), 0o644); err != nil {
			return "", err
		}
		return ActionApplied, nil
	}
	if userCert.Match(body) {
		return ActionSkipped, nil
	}
	var out []byte
	if i := bytes.Index(body, trustClose); i >= 0 {
		out = append(append([]byte{}, body[:i]...), append(userCertXML, body[i:]...)...)
	} else if loc := nscOpen.FindIndex(body); loc != nil {
		out = append(append([]byte{}, body[:loc[1]]...), append([]byte("\n"+baseConfigBlock), body[loc[1]:]...)...)
	} else {
		if err := os.WriteFile(path, []byte(defaultNSC), 0o644); err != nil {
			return "", err
		}
		return ActionApplied, nil
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", err
	}
	return ActionApplied, nil
}

func nscPath(dir, name string) string {
	return filepath.Join(dir, "res", "xml", name+".xml")
}

func applyNSC(dir string, tag []byte) (newTag []byte, action Action, err error) {
	tag, name, _ := setNSCAttr(tag)
	action, err = ensureNSC(nscPath(dir, name))
	if err != nil {
		return nil, "", fmt.Errorf("nsc: %w", err)
	}
	return tag, action, nil
}
