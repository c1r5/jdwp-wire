package device

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

const psToybox = `USER            PID  PPID     VSZ    RSS WCHAN            ADDR S NAME
u0_a58         4322     1 1234567  12345 SyS_epoll_wait      0 S com.alvo:remote
u0_a58         4321     1 1234567  12345 SyS_epoll_wait      0 S com.alvo
u0_a58         4400     1 1234567  12345 SyS_epoll_wait      0 S com.alvo:id
u0_a99         5000     1 1234567  12345 SyS_epoll_wait      0 S com.alvo.other
system         1000     1    1000    100                 0   0 S zygote
`

const psBusybox = `  PID USER       TIME COMMAND
 4321 u0_a58     0:00 com.alvo
 4400 u0_a58     0:00 com.alvo:id
`

func TestParsePidof(t *testing.T) {
	t.Parallel()
	got := parsePidof("4321 4400\n")
	if len(got) != 2 || got[0] != 4321 || got[1] != 4400 {
		t.Fatalf("got %#v", got)
	}
	if len(parsePidof("")) != 0 {
		t.Fatal("empty pidof should be empty")
	}
}

func TestParsePSToybox(t *testing.T) {
	t.Parallel()
	got := parsePS(psToybox, "com.alvo")
	want := []Process{
		{PID: 4322, Package: "com.alvo:remote"},
		{PID: 4321, Package: "com.alvo"},
		{PID: 4400, Package: "com.alvo:id"},
	}
	assertProcs(t, got, want)
}

func TestParsePSBusybox(t *testing.T) {
	t.Parallel()
	got := parsePS(psBusybox, "com.alvo")
	want := []Process{
		{PID: 4321, Package: "com.alvo"},
		{PID: 4400, Package: "com.alvo:id"},
	}
	assertProcs(t, got, want)
}

func TestMergeProcessesOrder(t *testing.T) {
	t.Parallel()
	got := mergeProcesses("com.alvo",
		[]int{4321, 4500},
		[]Process{
			{PID: 4400, Package: "com.alvo:id"},
			{PID: 4322, Package: "com.alvo:remote"},
			{PID: 4321, Package: "com.alvo"},
		},
	)
	want := []Process{
		{PID: 4321, Package: "com.alvo"},
		{PID: 4500, Package: "com.alvo"},
		{PID: 4322, Package: "com.alvo:remote"},
		{PID: 4400, Package: "com.alvo:id"},
	}
	assertProcs(t, got, want)
}

func TestADBPidofHybrid(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		if name != "adb" || len(args) < 4 || args[0] != "-s" || args[1] != "emulator-5554" || args[2] != "shell" {
			t.Fatalf("unexpected %s %v", name, args)
		}
		switch args[3] {
		case "pidof":
			if len(args) != 5 || args[4] != "com.alvo" {
				t.Fatalf("pidof args %v", args)
			}
			return execx.Result{Stdout: "4321 4500\n"}, nil
		case "ps":
			if len(args) != 5 || args[4] != "-A" {
				t.Fatalf("ps args %v", args)
			}
			return execx.Result{Stdout: psToybox}, nil
		default:
			t.Fatalf("unexpected shell %v", args)
			return execx.Result{}, nil
		}
	}}
	got, err := NewADB(r).Pidof(withTimeout(t), "emulator-5554", "com.alvo")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := []Process{
		{PID: 4321, Package: "com.alvo"},
		{PID: 4500, Package: "com.alvo"},
		{PID: 4322, Package: "com.alvo:remote"},
		{PID: 4400, Package: "com.alvo:id"},
	}
	assertProcs(t, got, want)
}

func TestADBPidofDeadApp(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		if args[3] == "pidof" {
			return execx.Result{ExitCode: 1}, execx.ErrExit
		}
		return execx.Result{Stdout: psToybox}, nil
	}}
	got, err := NewADB(r).Pidof(withTimeout(t), "emu", "com.missing")
	if err != nil {
		t.Fatalf("dead app should not error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}

func TestADBPidofUsage(t *testing.T) {
	t.Parallel()
	a := NewADB(&execx.Fake{})
	_, err := a.Pidof(withTimeout(t), "", "com.alvo")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("empty serial: %v", err)
	}
	_, err = a.Pidof(withTimeout(t), "emu", "")
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("empty pkg: %v", err)
	}
}

func TestADBPidofToolMissing(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{}, execx.ErrNotFound
	}}
	_, err := NewADB(r).Pidof(withTimeout(t), "emu", "com.alvo")
	if !errors.Is(err, ErrToolMissing) {
		t.Fatalf("err = %v, want ErrToolMissing", err)
	}
	if err == nil || !strings.Contains(err.Error(), "device: pidof:") {
		t.Fatalf("err = %v, want device: pidof: prefix", err)
	}
}

func assertProcs(t *testing.T, got, want []Process) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d\ngot  %#v\nwant %#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("i=%d got %#v, want %#v", i, got[i], want[i])
		}
	}
}

var _ Client = (*ADB)(nil)
