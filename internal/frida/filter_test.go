package frida

import (
	"strings"
	"testing"
)

const fridaBanner = `
     ____
    / _  |   Frida 16.5.6 - A world-class dynamic instrumentation toolkit
   | (_| |
    > _  |   Commands:
   /_/ |_|       help      -> Displays the help system
   . . . .       object?   -> Display information about 'object'
   . . . .       exit/quit -> Exit
   . . . .
   . . . .   More info at https://frida.re/docs/home/
   . . . .
   . . . .   Connected to Android Emulator 5554 (id=emulator-5554)
Attaching...
[Android::com.alvo]->
[Android::com.alvo]-> sslpinning-bypass armed
`

func TestFilter_DropsBanner(t *testing.T) {
	t.Parallel()
	f := NewFilter()
	var kept []string
	for _, line := range strings.Split(fridaBanner, "\n") {
		out, ok := f.Line(line)
		if ok {
			kept = append(kept, out)
		}
	}
	if len(kept) != 1 || kept[0] != "sslpinning-bypass armed" {
		t.Fatalf("kept = %#v", kept)
	}
}

func TestRewrite_StripsPrompt(t *testing.T) {
	t.Parallel()
	if got := rewrite("[Android::com.alvo]-> hooked\r"); got != "hooked" {
		t.Fatalf("got %q", got)
	}
	if got := rewrite("   \t"); got != "" {
		t.Fatalf("blank %q", got)
	}
}
