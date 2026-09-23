package attach

import (
	"context"
	"errors"
	"fmt"

	"github.com/c1r5/jdwp-wire/internal/device"
	"github.com/c1r5/jdwp-wire/internal/jdwp"
	"github.com/c1r5/jdwp-wire/internal/project"
	"github.com/c1r5/jdwp-wire/internal/workspace"
)

// StudioStatus is what attach did with the IntelliJ skeleton.
type StudioStatus int

const (
	StudioOff StudioStatus = iota
	StudioWritten
	StudioNoDecode
)

type studioWriter interface {
	Write(cfg project.Config) (project.Result, error)
}

var (
	_ studioWriter = project.FS{}
	_ studioWriter = (*project.Fake)(nil)
)

// Options is the attach request. Writer is used only when Studio is set.
type Options struct {
	Serial  string
	Package string
	Port    int
	CWD     string
	Studio  bool
	Writer  studioWriter
}

// Result is the JDWP session plus whether .jdt/<pkg>/idea was written.
type Result struct {
	Session jdwp.Session
	Studio  StudioStatus
	Project project.Result
}

// Run resolves the device, forwards JDWP, and with Studio writes the IntelliJ skeleton.
func Run(ctx context.Context, dev device.Client, client jdwp.Client, opt Options) (Result, error) {
	var layout workspace.Layout
	if opt.Studio {
		var err error
		layout, err = workspace.ForPackage(opt.CWD, opt.Package)
		if err != nil {
			return Result{}, fmt.Errorf("attach: %v: %w", err, jdwp.ErrUsage)
		}
		if opt.Writer == nil {
			return Result{}, fmt.Errorf("attach: studio: %w", project.ErrUsage)
		}
	}

	d, err := dev.Resolve(ctx, opt.Serial)
	if err != nil {
		return Result{}, err
	}
	sess, err := client.Attach(ctx, d.Serial, opt.Package, opt.Port)
	if err != nil {
		return Result{}, err
	}
	if !opt.Studio {
		return Result{Session: sess, Studio: StudioOff}, nil
	}

	wrote, err := opt.Writer.Write(project.Config{Layout: layout, Port: sess.Port})
	if errors.Is(err, project.ErrNoDecode) {
		return Result{Session: sess, Studio: StudioNoDecode}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("attach: studio: %w", err)
	}
	return Result{Session: sess, Studio: StudioWritten, Project: wrote}, nil
}
