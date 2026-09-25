package frida

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestOpen_FollowStripsBanner(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	body := fridaBanner + "\n"
	proc := holdProc(body)
	var args []string
	opened, srv, err := Open(context.Background(), Deps{
		Run:  runningServer(),
		Now:  func() time.Time { return time.Date(2026, 9, 23, 1, 0, 0, 0, time.UTC) },
		Look: func(string) (string, error) { return "frida", nil },
		Start: func(_ context.Context, name string, a ...string) (Proc, error) {
			args = append([]string{name}, a...)
			return proc, nil
		},
	}, "emulator-5554", "com.alvo", 7, 8700, dir, []Ref{{Name: NameSSL}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if srv.Started || srv.PID != 9 {
		t.Fatalf("server %+v", srv)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-D emulator-5554") || !strings.Contains(joined, "-p 7") || strings.Count(joined, " -l ") != 1 || !strings.Contains(joined, "load.js") {
		t.Fatalf("args %s", joined)
	}
	loader, err := os.ReadFile(dir + "/load.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(loader), "Application.attach") || !strings.Contains(string(loader), "sslpinning-bypass armed") {
		t.Fatalf("loader %s", loader)
	}
	if strings.Contains(joined, "--eternalize") || strings.Contains(joined, " -q") {
		t.Fatalf("args %s", joined)
	}
	text, err := osRead(opened.Log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "Frida 16") || strings.TrimSpace(text) != "sslpinning-bypass armed" {
		t.Fatalf("log %q", text)
	}
	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- opened.Follow(context.Background(), &buf) }()
	proc.release(nil)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "sslpinning-bypass armed" {
		t.Fatalf("follow %q", buf.String())
	}
}

func TestOpen_LoadFailure(t *testing.T) {
	t.Parallel()
	proc := holdProc("Failed to attach: unable to connect\n")
	_, _, err := Open(context.Background(), Deps{
		Run:  runningServer(),
		Look: func(string) (string, error) { return "frida", nil },
		Start: func(context.Context, string, ...string) (Proc, error) {
			return proc, nil
		},
	}, "emulator-5554", "com.alvo", 7, 8700, t.TempDir(), []Ref{{Name: NameRoot}}, false)
	if err == nil || !strings.Contains(err.Error(), "Failed to attach") {
		t.Fatalf("err %v", err)
	}
}

func TestOpen_Detach(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var got []string
	r, w := io.Pipe()
	opened, _, err := Open(context.Background(), Deps{
		Run:  runningServer(),
		Now:  func() time.Time { return time.Date(2026, 9, 23, 1, 0, 0, 0, time.UTC) },
		Look: func(string) (string, error) { return "frida", nil },
		Spawn: func(name string, args ...string) (int, io.ReadCloser, error) {
			got = append([]string{name}, args...)
			go func() { _, _ = fmt.Fprintln(w, "ready") }()
			return 4242, r, nil
		},
	}, "emulator-5554", "com.alvo", 7, 8700, dir, []Ref{{Name: NameSSL}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !opened.Detach || opened.Follow(context.Background(), io.Discard) != nil {
		t.Fatalf("%+v", opened)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "frida-session") || !strings.Contains(joined, "--serial emulator-5554") || !strings.Contains(joined, "--pid 7") || !strings.Contains(joined, "--package com.alvo") || !strings.Contains(joined, "--port 8700") || !strings.Contains(joined, "load.js") {
		t.Fatalf("spawn %s", joined)
	}
	body, err := osRead(dir + "/session.pid")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(body) != "4242" {
		t.Fatalf("pid %q", body)
	}
}

func TestStopSession_SignalsPID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	if err := writePID(dir, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	stopped, err := StopSession(dir)
	if err != nil || !stopped {
		t.Fatalf("stopped %v err %v", stopped, err)
	}
	_ = cmd.Wait()
	if _, err := osRead(dir + "/session.pid"); err == nil {
		t.Fatal("pid file remains")
	}
}

func TestOpen_StopKills(t *testing.T) {
	t.Parallel()
	proc := holdProc("sslpinning-bypass armed\n")
	opened, _, err := Open(context.Background(), Deps{
		Run:  runningServer(),
		Look: func(string) (string, error) { return "frida", nil },
		Start: func(context.Context, string, ...string) (Proc, error) {
			return proc, nil
		},
	}, "emulator-5554", "com.alvo", 7, 8700, t.TempDir(), []Ref{{Name: NameSSL}}, false)
	if err != nil {
		t.Fatal(err)
	}
	opened.Stop()
	select {
	case <-proc.killed:
	case <-time.After(time.Second):
		t.Fatal("frida still running")
	}
}

func TestServe_CancelClosesApp(t *testing.T) {
	t.Parallel()
	proc := holdProc("sslpinning-bypass armed\n")
	ctx, cancel := context.WithCancel(context.Background())
	var got []string
	statusR, statusW := io.Pipe()
	errc := make(chan error, 1)
	go func() {
		errc <- Serve(ctx, Deps{
			Look: func(string) (string, error) { return "frida", nil },
			Start: func(context.Context, string, ...string) (Proc, error) {
				return proc, nil
			},
			Run: &execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
				got = append(got, strings.Join(append([]string{name}, args...), " "))
				return execx.Result{}, nil
			}},
		}, "emulator-5554", "com.alvo", 7, 8700, filepath.Join(t.TempDir(), "frida.log"), []string{"/x.js"}, statusW)
	}()
	line, readErr := bufio.NewReader(statusR).ReadString('\n')
	if readErr != nil || !strings.Contains(line, "ready") {
		t.Fatalf("status %q err %v", line, readErr)
	}
	cancel()
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	select {
	case <-proc.killed:
	default:
		t.Fatal("frida still running")
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"am force-stop com.alvo", "am clear-debug-app", "forward --remove tcp:8700"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in %s", want, joined)
		}
	}
}

func runningServer() execx.Runner {
	return &execx.Fake{RunFn: func(context.Context, string, ...string) (execx.Result, error) {
		return execx.Result{Stdout: "9\n"}, nil
	}}
}

type hold struct {
	r      *io.PipeReader
	w      *io.PipeWriter
	exit   chan error
	killed chan struct{}
}

func holdProc(text string) *hold {
	r, w := io.Pipe()
	h := &hold{r: r, w: w, exit: make(chan error, 1), killed: make(chan struct{})}
	go func() { _, _ = io.WriteString(w, text) }()
	return h
}

func (h *hold) PID() int          { return 1 }
func (h *hold) Stdout() io.Reader { return h.r }
func (h *hold) Stderr() io.Reader { return strings.NewReader("") }
func (h *hold) Kill() error {
	select {
	case <-h.killed:
	default:
		close(h.killed)
	}
	_ = h.w.Close()
	return nil
}
func (h *hold) Wait() error {
	select {
	case <-h.killed:
		return fmt.Errorf("killed")
	case err := <-h.exit:
		return err
	}
}
func (h *hold) release(err error) {
	_ = h.w.Close()
	h.exit <- err
}

func osRead(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}
