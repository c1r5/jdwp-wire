package device

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestParseDebuggable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		out  string
		ok   bool
		err  error
	}{
		{
			name: "pkgFlags",
			out:  "    pkgFlags=[ DEBUGGABLE HAS_CODE ALLOW_CLEAR_USER_DATA ]\n",
			ok:   true,
		},
		{
			name: "flags line",
			out:  "    flags=[ HAS_CODE DEBUGGABLE ]\n",
			ok:   true,
		},
		{
			name: "release",
			out:  "    flags=[ SYSTEM HAS_CODE ALLOW_CLEAR_USER_DATA ]\n    pkgFlags=[ SYSTEM HAS_CODE ALLOW_CLEAR_USER_DATA ]\n",
		},
		{
			name: "permission flags",
			out:  "        android.permission.CAMERA: granted=true, flags=[ SYSTEM_FIXED|GRANTED_BY_DEFAULT ]\n",
		},
		{
			name: "missing",
			out:  "Unable to find package: com.alvo\n",
			err:  ErrPackageNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok, err := parseDebuggable(tc.out, "com.alvo")
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil || ok != tc.ok {
				t.Fatalf("ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestADBDebuggable(t *testing.T) {
	t.Parallel()
	var got []string
	a := NewADB(&execx.Fake{RunFn: func(_ context.Context, name string, args ...string) (execx.Result, error) {
		got = append([]string{name}, args...)
		return execx.Result{Stdout: "    pkgFlags=[ DEBUGGABLE HAS_CODE ]\n"}, nil
	}})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ok, err := a.Debuggable(ctx, "emu", "com.alvo")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if strings.Join(got, " ") != "adb -s emu shell dumpsys package com.alvo" {
		t.Fatalf("cmd %v", got)
	}
}
