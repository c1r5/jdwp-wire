package jdwp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
)

type pidSeq struct {
	n   int
	seq [][]device.Process
}

func (p *pidSeq) List(context.Context) ([]device.Device, error) { return nil, nil }
func (p *pidSeq) Resolve(context.Context, string) (device.Device, error) {
	return device.Device{}, nil
}
func (p *pidSeq) ListApps(context.Context, string) ([]device.App, error) { return nil, nil }
func (p *pidSeq) Pidof(_ context.Context, _, _ string) ([]device.Process, error) {
	i := p.n
	if i >= len(p.seq) {
		i = len(p.seq) - 1
	}
	p.n++
	src := p.seq[i]
	out := make([]device.Process, len(src))
	copy(out, src)
	return out, nil
}

func TestWaitPIDReady(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		if strings.Contains(strings.Join(args, " "), "jdwp") {
			return execx.Result{Stdout: "1234\n"}, nil
		}
		return execx.Result{}, errors.New("unexpected " + strings.Join(args, " "))
	}}
	a := New(r, &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 1234, Package: "com.alvo"}},
	}})
	a.poll = 0
	pid, err := a.waitPID(withTimeout(t), "emu", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if pid != 1234 {
		t.Fatalf("pid %d", pid)
	}
}

func TestWaitPIDIgnoresIsolated(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{Stdout: "11\n"}, nil
	}}
	a := New(r, &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 11, Package: "com.alvo:id"}},
	}})
	a.poll = 0
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := a.waitPID(ctx, "emu", "com.alvo")
	if !errors.Is(err, ErrNoProcess) {
		t.Fatalf("err=%v", err)
	}
}

func TestWaitPIDAppearsOnLaterPoll(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		return execx.Result{Stdout: "99\n"}, nil
	}}
	a := New(r, &pidSeq{seq: [][]device.Process{
		nil,
		{{PID: 99, Package: "com.alvo"}},
	}})
	a.poll = 0
	pid, err := a.waitPID(withTimeout(t), "emu", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if pid != 99 {
		t.Fatalf("pid %d", pid)
	}
}

func TestWaitPIDRequiresJDWPList(t *testing.T) {
	t.Parallel()
	n := 0
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		n++
		if n == 1 {
			return execx.Result{Stdout: ""}, nil
		}
		return execx.Result{Stdout: "7\n"}, nil
	}}
	a := New(r, &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 7, Package: "com.alvo"}},
	}})
	a.poll = 0
	pid, err := a.waitPID(withTimeout(t), "emu", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if pid != 7 {
		t.Fatalf("pid %d", pid)
	}
}

func TestLaunchResolveExitFallsBackToMonkey(t *testing.T) {
	t.Parallel()
	var last []string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		last = append([]string{name}, args...)
		if strings.Contains(strings.Join(args, " "), "resolve-activity") {
			return execx.Result{Stderr: "No activity found\n"}, &execx.ExitError{Name: "adb", ExitCode: 1, Stderr: "No activity found\n"}
		}
		return execx.Result{}, nil
	}}
	act, err := New(r, &device.Fake{}).launch(withTimeout(t), "emu", "com.alvo")
	if err != nil {
		t.Fatal(err)
	}
	if act != "" {
		t.Fatalf("activity %q", act)
	}
	if last[4] != "monkey" {
		t.Fatalf("last %v", last)
	}
}
