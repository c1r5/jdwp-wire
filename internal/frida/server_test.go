package frida

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestEnsureServer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		reply   func(args []string, ctx context.Context) (execx.Result, error)
		wantPID int
		started bool
		wantErr string
		tool    bool
	}{
		{
			name: "already",
			reply: func(args []string, _ context.Context) (execx.Result, error) {
				if strings.Contains(strings.Join(args, " "), "su -c") {
					t.Fatal("su called")
				}
				return execx.Result{Stdout: "9\n"}, nil
			},
			wantPID: 9,
		},
		{
			name: "missing binary",
			reply: func(args []string, _ context.Context) (execx.Result, error) {
				joined := strings.Join(args, " ")
				if strings.Contains(joined, "su -c") {
					t.Fatal("su called")
				}
				if strings.Contains(joined, "pidof") {
					return execx.Result{ExitCode: 1}, &execx.ExitError{Name: "adb", ExitCode: 1}
				}
				return execx.Result{ExitCode: 1, Stderr: "No such file"}, &execx.ExitError{Name: "adb", ExitCode: 1, Stderr: "No such file"}
			},
			wantErr: serverRemote + " missing",
		},
		{
			name: "starts",
			reply: func(args []string, ctx context.Context) (execx.Result, error) {
				joined := strings.Join(args, " ")
				switch {
				case strings.Contains(joined, "su -c"):
					dl, ok := ctx.Deadline()
					if !ok || time.Until(dl) > serverBoot+200*time.Millisecond {
						t.Fatalf("launch deadline %v ok=%v", time.Until(dl), ok)
					}
					if !strings.Contains(joined, "sleep 2147483647") {
						t.Fatalf("launch %s", joined)
					}
					return execx.Result{}, context.DeadlineExceeded
				case strings.Contains(joined, "pidof"):
					if strings.Count(joined, "pidof") > 0 && pidofCalls == 0 {
						pidofCalls++
						return execx.Result{ExitCode: 1}, &execx.ExitError{Name: "adb", ExitCode: 1}
					}
					return execx.Result{Stdout: "42\n"}, nil
				default:
					return execx.Result{Stdout: serverRemote + "\n"}, nil
				}
			},
			wantPID: 42,
			started: true,
		},
		{
			name: "su missing",
			reply: func(args []string, _ context.Context) (execx.Result, error) {
				joined := strings.Join(args, " ")
				if strings.Contains(joined, "su -c") {
					return execx.Result{ExitCode: 127, Stderr: "su: not found"}, &execx.ExitError{Name: "adb", ExitCode: 127, Stderr: "su: not found"}
				}
				if strings.Contains(joined, "pidof") {
					return execx.Result{ExitCode: 1}, &execx.ExitError{Name: "adb", ExitCode: 1}
				}
				return execx.Result{}, nil
			},
			wantErr: "su: not found",
		},
		{
			name: "adb missing",
			reply: func([]string, context.Context) (execx.Result, error) {
				return execx.Result{}, execx.ErrNotFound
			},
			tool: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pidofCalls = 0
			fake := &execx.Fake{RunFn: func(ctx context.Context, name string, args ...string) (execx.Result, error) {
				if name != "adb" {
					t.Fatalf("name %s", name)
				}
				return tc.reply(append([]string{name}, args...), ctx)
			}}
			got, err := EnsureServer(context.Background(), fake, "emulator-5554")
			if tc.tool {
				if !errors.Is(err, ErrToolMissing) {
					t.Fatalf("err %v", err)
				}
				return
			}
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.PID != tc.wantPID || got.Started != tc.started {
				t.Fatalf("%+v", got)
			}
		})
	}
}

var pidofCalls int
