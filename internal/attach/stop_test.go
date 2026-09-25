package attach

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/jdwp"
)

func TestRun_AttachErrorStaysLaunched(t *testing.T) {
	t.Parallel()
	res, err := Run(context.Background(), testDevice(), &jdwp.Fake{
		AttachFn: func(context.Context, string, string, int) (jdwp.Session, error) {
			return jdwp.Session{}, errors.New("bind")
		},
	}, Options{
		Package: "com.alvo",
		Port:    8700,
		CWD:     t.TempDir(),
		APK:     adbTools(t),
		Patch:   appliedPatch(),
	})
	if err == nil || !res.Launched {
		t.Fatalf("launched %v err %v", res.Launched, err)
	}
	if res.Session.Serial != "emulator-5554" || res.Session.Package != "com.alvo" || res.Session.Port != 8700 {
		t.Fatalf("%+v", res.Session)
	}
}

func TestStop_ClosesLaunchedApp(t *testing.T) {
	t.Parallel()
	var got string
	err := Stop(&jdwp.Fake{ForceStopFn: func(serial, pkg string, port int) error {
		got = serial + " " + pkg + " " + strconv.Itoa(port)
		return nil
	}}, Result{
		Launched: true,
		Session:  jdwp.Session{Serial: "emulator-5554", Package: "com.alvo", Port: 8700},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "emulator-5554 com.alvo 8700" {
		t.Fatalf("stop %q", got)
	}
}

func TestStop_SkipsBeforeLaunch(t *testing.T) {
	t.Parallel()
	err := Stop(&jdwp.Fake{ForceStopFn: func(string, string, int) error {
		t.Fatal("force-stop")
		return nil
	}}, Result{})
	if err != nil {
		t.Fatal(err)
	}
}
