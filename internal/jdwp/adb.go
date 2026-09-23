package jdwp

import (
	"time"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
	"github.com/c1r5/jdwp-wire/internal/logging"
)

type ADB struct {
	r       execx.Runner
	d       device.Client
	poll    time.Duration
	listFor time.Duration
	log     *logging.Logger
}

// SetLog prints each JDWP step as it finishes. Nil stays quiet.
func (a *ADB) SetLog(l *logging.Logger) {
	if a == nil {
		return
	}
	a.log = l
}

func New(r execx.Runner, d device.Client) *ADB {
	return &ADB{r: r, d: d, poll: 200 * time.Millisecond, listFor: 2 * time.Second}
}

var _ Client = (*ADB)(nil)
