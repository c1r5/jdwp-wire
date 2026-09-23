package jdwp

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
)

func listenLocal(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_, _ = io.Copy(io.Discard, c)
			}(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func closedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func attachRunner(t *testing.T, cmds *[][]string) execx.Runner {
	t.Helper()
	return commandRunner(t, cmds, true)
}

func commandRunner(t *testing.T, cmds *[][]string, list bool) execx.Runner {
	t.Helper()
	return &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		*cmds = append(*cmds, append([]string{name}, args...))
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "resolve-activity"):
			return execx.Result{Stdout: "com.alvo/.MainActivity\n"}, nil
		case len(args) >= 3 && args[2] == "jdwp":
			return execx.Result{Stdout: "4242\n"}, nil
		case strings.Contains(joined, "--list") && list:
			return execx.Result{Stdout: forwardList(*cmds)}, nil
		default:
			return execx.Result{}, nil
		}
	}}
}

func forwardList(cmds [][]string) string {
	for i := len(cmds) - 1; i >= 0; i-- {
		args := cmds[i]
		serial := ""
		for k := 0; k < len(args)-1; k++ {
			if args[k] == "-s" {
				serial = args[k+1]
			}
		}
		for j := 0; j < len(args)-2; j++ {
			if args[j] == "forward" && strings.HasPrefix(args[j+1], "tcp:") && strings.HasPrefix(args[j+2], "jdwp:") {
				return serial + " " + args[j+1] + " " + args[j+2] + "\n"
			}
		}
	}
	return ""
}

func TestBindSkipsLaunch(t *testing.T) {
	t.Parallel()
	port := listenLocal(t)
	var cmds [][]string
	a := New(attachRunner(t, &cmds), &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 4242, Package: "com.alvo"}},
	}})
	a.poll = 0
	got, err := a.Bind(withTimeout(t), "emu", "com.alvo", port)
	if err != nil {
		t.Fatal(err)
	}
	if got.Activity != "" || got.PID != 4242 || got.Port != port {
		t.Fatalf("%+v", got)
	}
	for _, c := range cmds {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "set-debug-app") || strings.Contains(joined, "am start") || strings.Contains(joined, "monkey") {
			t.Fatalf("launch command %v", c)
		}
	}
}

func TestBindReadsStreamingJDWP(t *testing.T) {
	t.Parallel()
	port := listenLocal(t)
	a := New(&execx.Fake{RunFn: func(ctx context.Context, _ string, args ...string) (execx.Result, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--list") {
			return execx.Result{Stdout: "emu tcp:" + strconv.Itoa(port) + " jdwp:4242\n"}, nil
		}
		if len(args) >= 3 && args[2] == "jdwp" {
			<-ctx.Done()
			return execx.Result{Stdout: "4242\n"}, ctx.Err()
		}
		return execx.Result{}, nil
	}}, &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 4242, Package: "com.alvo"}},
	}})
	a.poll = 0
	a.listFor = 20 * time.Millisecond
	got, err := a.Bind(withTimeout(t), "emu", "com.alvo", port)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != 4242 || got.Port != port {
		t.Fatalf("%+v", got)
	}
}

func TestAttachSession(t *testing.T) {
	t.Parallel()
	port := listenLocal(t)
	var cmds [][]string
	a := New(attachRunner(t, &cmds), &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 4242, Package: "com.alvo"}},
	}})
	a.poll = 0
	got, err := a.Attach(withTimeout(t), "emu", "com.alvo", port)
	if err != nil {
		t.Fatal(err)
	}
	want := Session{
		Package:  "com.alvo",
		Serial:   "emu",
		PID:      4242,
		Port:     port,
		Activity: "com.alvo/.MainActivity",
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	fwd := []string{"adb", "-s", "emu", "forward", "tcp:" + strconv.Itoa(port), "jdwp:4242"}
	var sawForward bool
	for _, c := range cmds {
		if reflect.DeepEqual(c, fwd) {
			sawForward = true
		}
	}
	if !sawForward {
		t.Fatalf("missing %v in %v", fwd, cmds)
	}
}

func TestAttachUsage(t *testing.T) {
	t.Parallel()
	a := New(&execx.Fake{}, &device.Fake{})
	ctx := withTimeout(t)
	cases := []struct {
		serial string
		pkg    string
		port   int
	}{
		{"", "com.alvo", 8700},
		{"emu", "", 8700},
		{"emu", "com.alvo", 0},
		{"emu", "com.alvo", 70000},
	}
	for _, tc := range cases {
		_, err := a.Attach(ctx, tc.serial, tc.pkg, tc.port)
		if !errors.Is(err, ErrUsage) {
			t.Fatalf("%+v err=%v", tc, err)
		}
	}
}

func TestAttachProbeFails(t *testing.T) {
	t.Parallel()
	var cmds [][]string
	a := New(commandRunner(t, &cmds, false), &device.Fake{Procs: map[string][]device.Process{
		"com.alvo": {{PID: 4242, Package: "com.alvo"}},
	}})
	a.poll = 0
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := a.Attach(ctx, "emu", "com.alvo", closedPort(t))
	if !errors.Is(err, ErrProbe) {
		t.Fatalf("err=%v", err)
	}
}

func TestResetClearsAndRemovesForward(t *testing.T) {
	t.Parallel()
	var cmds [][]string
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		cmds = append(cmds, append([]string{name}, args...))
		return execx.Result{}, nil
	}}
	if err := New(r, &device.Fake{}).Reset(withTimeout(t), "emu", "com.alvo", 8700); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"adb", "-s", "emu", "shell", "am", "clear-debug-app"},
		{"adb", "-s", "emu", "forward", "--remove", "tcp:8700"},
	}
	if !reflect.DeepEqual(cmds, want) {
		t.Fatalf("got %v", cmds)
	}
}

func TestResetRemoveExitIsSkip(t *testing.T) {
	t.Parallel()
	r := &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		if strings.Contains(strings.Join(args, " "), "--remove") {
			return execx.Result{Stderr: "not found"}, &execx.ExitError{Name: "adb", ExitCode: 1, Stderr: "not found"}
		}
		return execx.Result{}, nil
	}}
	if err := New(r, &device.Fake{}).Reset(withTimeout(t), "emu", "com.alvo", 8700); err != nil {
		t.Fatal(err)
	}
}

func TestResetUsage(t *testing.T) {
	t.Parallel()
	err := New(&execx.Fake{}, &device.Fake{}).Reset(withTimeout(t), "", "com.alvo", 8700)
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("err=%v", err)
	}
}
