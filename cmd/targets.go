package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/c1r5/jdwp-wire/internal/targets"
	"github.com/spf13/cobra"
)

func newTargetsCmd(cfg runConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "targets <pkg|decoded>",
		Short: "List HTTP call sites (OkHttp, Retrofit, HttpURLConnection) as class#method",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			httpOn, err := c.Flags().GetBool("http")
			if err != nil {
				return err
			}
			if !httpOn {
				return fmt.Errorf("%w: --http is the only mode in this version", targets.ErrUsage)
			}
			asJSON, err := c.Flags().GetBool("json")
			if err != nil {
				return err
			}
			dir, err := targets.DecodeDir(cfg.cwd, args[0])
			if err != nil {
				return err
			}
			res, err := targets.ScanHTTP(dir)
			if err != nil {
				return err
			}
			if abs, err := filepath.Abs(res.Dir); err == nil {
				res.Dir = abs
			}
			if asJSON {
				return writeTargetsJSON(c.OutOrStdout(), res)
			}
			return writeTargetsHuman(c.OutOrStdout(), res)
		},
	}
	c.Flags().Bool("http", true, "list HTTP sinks (OkHttp, Retrofit, HttpURLConnection)")
	return c
}

func writeTargetsHuman(w io.Writer, res targets.Result) error {
	if len(res.Hits) == 0 {
		_, err := fmt.Fprintln(w, "[skip] targets: no http sinks")
		return err
	}
	for _, h := range res.Hits {
		if _, err := fmt.Fprintf(w, "%s#%s\n", h.Class, h.Method); err != nil {
			return err
		}
	}
	return nil
}

type targetJSON struct {
	Class  string   `json:"class"`
	Method string   `json:"method"`
	Kinds  []string `json:"kinds"`
}

type targetsJSON struct {
	Dir     string       `json:"dir"`
	Targets []targetJSON `json:"targets"`
}

func writeTargetsJSON(w io.Writer, res targets.Result) error {
	out := targetsJSON{Dir: res.Dir, Targets: make([]targetJSON, 0, len(res.Hits))}
	for _, h := range res.Hits {
		kinds := make([]string, len(h.Kinds))
		for i, k := range h.Kinds {
			kinds[i] = string(k)
		}
		out.Targets = append(out.Targets, targetJSON{Class: h.Class, Method: h.Method, Kinds: kinds})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
