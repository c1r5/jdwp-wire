package project

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed testdata/decode.iml testdata/misc.xml testdata/modules.xml testdata/Remote_Debug.xml
var templates embed.FS

const defaultPortOption = `<option name="PORT" value="8700" />`

var _ Writer = FS{}

func (FS) Write(cfg Config) (Result, error) {
	if cfg.Layout.Idea == "" || cfg.Layout.Decode == "" {
		return Result{}, fmt.Errorf("project: write: %w", ErrUsage)
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Result{}, fmt.Errorf("project: write: %w", ErrUsage)
	}
	if err := requireDecode(cfg.Layout.Decode); err != nil {
		return Result{}, fmt.Errorf("project: write: %w", err)
	}

	idea := cfg.Layout.Idea
	iml := filepath.Join(idea, "decode.iml")
	misc := filepath.Join(idea, ".idea", "misc.xml")
	modules := filepath.Join(idea, ".idea", "modules.xml")
	run := filepath.Join(idea, ".idea", "runConfigurations", "Remote_Debug.xml")
	if err := os.MkdirAll(filepath.Dir(run), 0o755); err != nil {
		return Result{}, fmt.Errorf("project: write: %w", err)
	}

	static := []struct{ dest, name string }{
		{iml, "testdata/decode.iml"},
		{misc, "testdata/misc.xml"},
		{modules, "testdata/modules.xml"},
	}
	for _, f := range static {
		body, err := templates.ReadFile(f.name)
		if err != nil {
			return Result{}, fmt.Errorf("project: write: %w", err)
		}
		if err := os.WriteFile(f.dest, body, 0o644); err != nil {
			return Result{}, fmt.Errorf("project: write: %w", err)
		}
	}

	raw, err := templates.ReadFile("testdata/Remote_Debug.xml")
	if err != nil {
		return Result{}, fmt.Errorf("project: write: %w", err)
	}
	portOpt := fmt.Sprintf(`<option name="PORT" value="%d" />`, cfg.Port)
	body := strings.Replace(string(raw), defaultPortOption, portOpt, 1)
	if err := os.WriteFile(run, []byte(body), 0o644); err != nil {
		return Result{}, fmt.Errorf("project: write: %w", err)
	}

	return Result{
		Dir:       idea,
		Decode:    cfg.Layout.Decode,
		IML:       iml,
		RunConfig: run,
	}, nil
}

func requireDecode(dir string) error {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return ErrNoDecode
	}
	man, err := os.Stat(filepath.Join(dir, "AndroidManifest.xml"))
	if err != nil || man.IsDir() {
		return ErrNoDecode
	}
	return nil
}
