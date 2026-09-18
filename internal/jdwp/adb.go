package jdwp

import (
	"time"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/execx"
)

type ADB struct {
	r    execx.Runner
	d    device.Client
	poll time.Duration
}

func New(r execx.Runner, d device.Client) *ADB {
	return &ADB{r: r, d: d, poll: 200 * time.Millisecond}
}

var _ Client = (*ADB)(nil)
