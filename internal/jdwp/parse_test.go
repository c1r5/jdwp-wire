package jdwp

import (
	"reflect"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/device"
)

func TestParseActivityBrief(t *testing.T) {
	t.Parallel()
	got := parseActivity("com.alvo/.MainActivity\n")
	if got != "com.alvo/.MainActivity" {
		t.Fatalf("got %q", got)
	}
}

func TestParseActivitySkipsNoise(t *testing.T) {
	t.Parallel()
	in := "priority=0 preferredOrder=0 match=0x108000 specificIndex=-1 isDefault=false\ncom.alvo/.MainActivity\n"
	got := parseActivity(in)
	if got != "com.alvo/.MainActivity" {
		t.Fatalf("got %q", got)
	}
}

func TestParseActivityEmpty(t *testing.T) {
	t.Parallel()
	if parseActivity("") != "" {
		t.Fatal("want empty")
	}
	if parseActivity("No activity found\n") != "" {
		t.Fatal("want empty")
	}
}

func TestParseJDWP(t *testing.T) {
	t.Parallel()
	got := parseJDWP("1234\n5678\n")
	want := []int{1234, 5678}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestParseJDWPIgnoresJunk(t *testing.T) {
	t.Parallel()
	got := parseJDWP("* daemon started *\n\n42\nnot-a-pid\n")
	if !reflect.DeepEqual(got, []int{42}) {
		t.Fatalf("got %v", got)
	}
}

func TestPickPIDExactPackage(t *testing.T) {
	t.Parallel()
	procs := []device.Process{
		{PID: 10, Package: "com.alvo"},
		{PID: 11, Package: "com.alvo:remote"},
	}
	if pickPID("com.alvo", procs) != 10 {
		t.Fatalf("got %d", pickPID("com.alvo", procs))
	}
}

func TestPickPIDIgnoresIsolated(t *testing.T) {
	t.Parallel()
	procs := []device.Process{
		{PID: 11, Package: "com.alvo:id"},
		{PID: 12, Package: "com.alvo:remote"},
	}
	if pickPID("com.alvo", procs) != 0 {
		t.Fatal("isolated must not count")
	}
}
