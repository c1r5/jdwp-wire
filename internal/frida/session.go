package frida

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

const readyWait = 750 * time.Millisecond

// Proc is a running frida CLI. execx.Cmd is adapted by the caller.
type Proc interface {
	PID() int
	Stdout() io.Reader
	Stderr() io.Reader
	Wait() error
	Kill() error
}

// Deps are the process and clock hooks. Nil Start, Spawn, Look, and Now use execx and time.Now.
type Deps struct {
	Run        execx.Runner
	Start      func(context.Context, string, ...string) (Proc, error)
	Spawn      func(name string, args ...string) (int, io.ReadCloser, error)
	Look       func(string) (string, error)
	Now        func() time.Time
	ReadyAfter time.Duration
}

// Opened is a Frida session that is already past the banner.
// Follow blocks on the foreground path. Detach has no Follow.
type Opened struct {
	Names  []string
	Log    string
	Detach bool

	follow func(context.Context, io.Writer) error
}

// Follow copies rewritten lines to w until the CLI exits or ctx is canceled.
// Cancel is a clean stop. Detach returns nil.
func (o *Opened) Follow(ctx context.Context, w io.Writer) error {
	if o == nil || o.Detach || o.follow == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return o.follow(ctx, w)
}

// Open checks frida-server, writes the scripts, and starts the CLI.
// dir is .jdt/<pkg>/frida. The returned Server says whether this call started it.
func Open(ctx context.Context, dep Deps, serial string, pid int, dir string, refs []Ref, detach bool) (Opened, Server, error) {
	if pid <= 0 || serial == "" {
		return Opened{}, Server{}, fmt.Errorf("frida: session: %w", ErrUsage)
	}
	srv, err := EnsureServer(ctx, dep.Run, serial)
	if err != nil {
		return Opened{}, Server{}, err
	}
	files, err := Materialize(dir, refs)
	if err != nil {
		return Opened{}, srv, err
	}
	look := dep.Look
	if look == nil {
		look = execx.Look
	}
	if _, err := look("frida"); err != nil {
		return Opened{}, srv, fmt.Errorf("frida: %w", ErrToolMissing)
	}
	now := dep.Now
	if now == nil {
		now = time.Now
	}
	logPath := LogPath(dir, now())
	names := make([]string, len(files))
	args := []string{"-D", serial, "-p", strconv.Itoa(pid)}
	for i, ref := range files {
		names[i] = ref.Name
		args = append(args, "-l", ref.Path)
	}
	wait := dep.ReadyAfter
	if wait <= 0 {
		wait = readyWait
	}
	if detach {
		opened, err := spawnSupervisor(ctx, dep, serial, pid, logPath, files, names)
		return opened, srv, err
	}
	start := dep.Start
	if start == nil {
		start = startExec
	}
	proc, err := start(context.Background(), "frida", args...)
	if err != nil {
		return Opened{}, srv, fmt.Errorf("frida: %w", err)
	}
	lg, err := OpenLog(dir, now())
	if err != nil {
		_ = proc.Kill()
		return Opened{}, srv, err
	}
	loop := newLoop(proc, lg, true)
	if err := loop.waitReady(ctx, wait); err != nil {
		_ = proc.Kill()
		_ = lg.Close()
		return Opened{}, srv, err
	}
	return Opened{Names: names, Log: lg.Path(), follow: loop.follow}, srv, nil
}

func startExec(ctx context.Context, name string, args ...string) (Proc, error) {
	cmd, err := execx.Exec{}.Start(ctx, name, args...)
	if err != nil {
		return nil, err
	}
	return cmdProc{Cmd: cmd}, nil
}

type cmdProc struct{ Cmd *execx.Cmd }

func (c cmdProc) PID() int          { return c.Cmd.PID() }
func (c cmdProc) Stdout() io.Reader { return c.Cmd.Stdout }
func (c cmdProc) Stderr() io.Reader { return c.Cmd.Stderr }
func (c cmdProc) Wait() error       { return c.Cmd.Wait() }
func (c cmdProc) Kill() error       { return c.Cmd.Kill() }

type loop struct {
	proc *tracked
	log  *Log
	keep bool

	mu      sync.Mutex
	mirror  io.Writer
	pending []string
	filter  *Filter

	ready    chan struct{}
	fail     chan error
	finished chan error
	once     sync.Once
}

type tracked struct {
	Proc
	waitOnce sync.Once
	waitErr  error
}

func (t *tracked) Wait() error {
	t.waitOnce.Do(func() { t.waitErr = t.Proc.Wait() })
	return t.waitErr
}

func newLoop(proc Proc, lg *Log, keep bool) *loop {
	l := &loop{
		proc:     &tracked{Proc: proc},
		log:      lg,
		keep:     keep,
		filter:   NewFilter(),
		ready:    make(chan struct{}),
		fail:     make(chan error, 1),
		finished: make(chan error, 1),
	}
	go l.read()
	return l
}

func (l *loop) read() {
	sc := bufio.NewScanner(merge(l.proc.Stdout(), l.proc.Stderr()))
	for sc.Scan() {
		raw := sc.Text()
		if loadFailed(raw) {
			l.failOnce(fmt.Errorf("frida: %s", strings.TrimSpace(raw)))
			continue
		}
		out, ok := l.filter.Line(raw)
		if !ok {
			continue
		}
		if err := l.log.WriteLine(out); err != nil {
			l.failOnce(err)
		}
		l.mu.Lock()
		if l.mirror != nil {
			_, _ = fmt.Fprintln(l.mirror, out)
		} else if l.keep {
			l.pending = append(l.pending, out)
		}
		l.mu.Unlock()
		l.markReady()
	}
	err := l.proc.Wait()
	if sc.Err() != nil && err == nil {
		err = sc.Err()
	}
	l.finished <- err
}

func (l *loop) markReady() { l.once.Do(func() { close(l.ready) }) }

func (l *loop) failOnce(err error) {
	select {
	case l.fail <- err:
	default:
	}
}

func (l *loop) waitReady(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("frida: %w", ctx.Err())
	case err := <-l.fail:
		return err
	case err := <-l.finished:
		if err == nil {
			err = fmt.Errorf("frida: cli exited")
		}
		return fmt.Errorf("frida: %w", err)
	case <-l.ready:
		return nil
	case <-timer.C:
		return nil
	}
}

func (l *loop) follow(ctx context.Context, w io.Writer) error {
	l.mu.Lock()
	l.mirror = w
	pending := l.pending
	l.pending = nil
	l.mu.Unlock()
	for _, line := range pending {
		if w != nil {
			_, _ = fmt.Fprintln(w, line)
		}
	}
	select {
	case <-ctx.Done():
		_ = l.proc.Kill()
		<-l.finished
		_ = l.log.Close()
		return nil
	case err := <-l.finished:
		_ = l.log.Close()
		if err == nil {
			return nil
		}
		return fmt.Errorf("frida: %w", err)
	}
}

func loadFailed(line string) bool {
	l := strings.ToLower(line)
	for _, n := range []string{"failed to attach", "failed to spawn", "unable to connect", "unable to find"} {
		if strings.Contains(l, n) {
			return true
		}
	}
	return false
}

func merge(a, b io.Reader) io.Reader {
	if a == nil {
		a = strings.NewReader("")
	}
	if b == nil {
		b = strings.NewReader("")
	}
	pr, pw := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(2)
	pump := func(r io.Reader) {
		defer wg.Done()
		_, _ = io.Copy(pw, r)
	}
	go pump(a)
	go pump(b)
	go func() {
		wg.Wait()
		_ = pw.Close()
	}()
	return pr
}

func spawnSupervisor(ctx context.Context, dep Deps, serial string, pid int, logPath string, files []Ref, names []string) (Opened, error) {
	exe, err := os.Executable()
	if err != nil {
		return Opened{}, fmt.Errorf("frida: %w", err)
	}
	args := []string{"frida-session", "--serial", serial, "--pid", strconv.Itoa(pid), "--log", logPath}
	for _, ref := range files {
		args = append(args, "--script", ref.Path)
	}
	spawn := dep.Spawn
	if spawn == nil {
		spawn = execx.StartDetached
	}
	spid, stdout, err := spawn(exe, args...)
	if err != nil {
		return Opened{}, fmt.Errorf("frida: %w", err)
	}
	line, err := readStatus(ctx, stdout)
	if err != nil || strings.HasPrefix(line, "fail") {
		_ = killPID(spid)
		if err == nil {
			err = fmt.Errorf("%s", strings.TrimPrefix(line, "fail "))
		}
		return Opened{}, fmt.Errorf("frida: %w", err)
	}
	if err := writePID(filepath.Dir(logPath), spid); err != nil {
		_ = killPID(spid)
		return Opened{}, err
	}
	return Opened{Names: names, Log: logPath, Detach: true}, nil
}

func readStatus(ctx context.Context, r io.Reader) (string, error) {
	ch := make(chan string, 1)
	errc := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(r).ReadString('\n')
		if err != nil && line == "" {
			errc <- err
			return
		}
		ch <- strings.TrimRight(line, "\r\n")
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errc:
		return "", err
	case line := <-ch:
		return line, nil
	}
}

func writePID(dir string, pid int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("frida: session: %w", err)
	}
	path := pidPath(dir)
	body := strconv.Itoa(pid) + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("frida: session: %w", err)
	}
	return nil
}

func pidPath(dir string) string { return dir + string(os.PathSeparator) + "session.pid" }

// Serve runs inside the detached jdt frida-session process.
// It writes "ready" or "fail …" to status, then keeps appending the daily log.
func Serve(ctx context.Context, serial string, pid int, logPath string, scripts []string, status io.Writer) error {
	if serial == "" || pid <= 0 || logPath == "" || len(scripts) == 0 {
		return fmt.Errorf("frida: session: %w", ErrUsage)
	}
	if _, err := execx.Look("frida"); err != nil {
		_, _ = fmt.Fprintln(status, "fail frida not on PATH")
		return fmt.Errorf("frida: %w", ErrToolMissing)
	}
	args := []string{"-D", serial, "-p", strconv.Itoa(pid)}
	for _, path := range scripts {
		args = append(args, "-l", path)
	}
	proc, err := startExec(ctx, "frida", args...)
	if err != nil {
		_, _ = fmt.Fprintf(status, "fail %s\n", err.Error())
		return err
	}
	lg, err := OpenLogAt(logPath)
	if err != nil {
		_ = proc.Kill()
		_, _ = fmt.Fprintf(status, "fail %s\n", err.Error())
		return err
	}
	loop := newLoop(proc, lg, false)
	if err := loop.waitReady(ctx, readyWait); err != nil {
		_ = proc.Kill()
		_ = lg.Close()
		_, _ = fmt.Fprintf(status, "fail %s\n", err.Error())
		return err
	}
	if _, err := fmt.Fprintln(status, "ready"); err != nil {
		_ = proc.Kill()
		return err
	}
	<-loop.finished
	_ = lg.Close()
	return nil
}
