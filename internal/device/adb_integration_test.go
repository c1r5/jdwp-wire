//go:build integration

package device

import (
	"errors"
	"testing"

	"github.com/c1r5/jdwp-wire/internal/execx"
)

func TestADBListIntegration(t *testing.T) {
	got, err := NewADB(execx.Exec{}).List(withTimeout(t))
	if errors.Is(err, ErrToolMissing) {
		t.Skip("adb not installed")
	}
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) == 0 {
		t.Skip("no adb devices")
	}
	for _, d := range got {
		if d.Serial == "" {
			t.Fatalf("empty serial in %#v", d)
		}
	}
}
