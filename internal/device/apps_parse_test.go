package device

import "testing"

func TestParsePackageList(t *testing.T) {
	t.Parallel()
	got := parsePackageList("package:com.b\r\npackage:com.a\nnoise\npackage:com.a\npackage:\n")
	want := []string{"com.b", "com.a"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}

func TestParseAppLabels(t *testing.T) {
	t.Parallel()
	stdout := `
Package [com.zeta] (1a):
  labelRes=0x0 nonLocalizedLabel=Zeta App icon=0x0 banner=0x0
Package [com.alpha] (2b):
  labelRes=0x7f nonLocalizedLabel=null icon=0x1 banner=0x0
  Application Label: Alpha
Package [not a pkg]:
  nonLocalizedLabel=Nope icon=0x0
Package []:
  Application Label: Missing
`
	got := parseAppLabels(stdout)
	if got["com.zeta"] != "Zeta App" {
		t.Fatalf("zeta = %q", got["com.zeta"])
	}
	if got["com.alpha"] != "Alpha" {
		t.Fatalf("alpha = %q", got["com.alpha"])
	}
	if _, ok := got["not a pkg"]; ok {
		t.Fatalf("spacey header stored: %v", got)
	}
}

func TestPidForPackagePrefersMain(t *testing.T) {
	t.Parallel()
	procs := []Process{
		{PID: 222, Package: "com.zeta:push"},
		{PID: 400, Package: "com.zeta"},
		{PID: 111, Package: "com.zeta"},
		{PID: 50, Package: "com.other"},
	}
	if got := pidForPackage("com.zeta", procs); got != 111 {
		t.Fatalf("main pid = %d", got)
	}
	if got := pidForPackage("com.zeta", []Process{{PID: 9, Package: "com.zeta:push"}, {PID: 12, Package: "com.zeta:remote"}}); got != 9 {
		t.Fatalf("subprocess pid = %d", got)
	}
	if got := pidForPackage("com.nope", procs); got != 0 {
		t.Fatalf("missing pid = %d", got)
	}
}
