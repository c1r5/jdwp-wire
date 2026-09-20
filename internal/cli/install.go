package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/c1r5/jdwp-wire/internal/apk"
	"github.com/c1r5/jdwp-wire/internal/workspace"
	"github.com/spf13/cobra"
)

type installPrep struct {
	Base    string
	Splits  []string
	Encoded bool
}

func newInstallCmd(cfg runConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <apk|decoded_dir|pkg>",
		Short: "Rebuild a decoded tree if needed, sign debug, and adb install -r -d",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.apkTimeout)
			defer cancel()
			serial, err := cmd.Flags().GetString("serial")
			if err != nil {
				return err
			}
			pkgFlag, err := cmd.Flags().GetString("package")
			if err != nil {
				return err
			}
			asJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			dev, err := cfg.device.Resolve(ctx, serial)
			if err != nil {
				return err
			}
			prep, err := prepareInstall(ctx, cfg, args[0], pkgFlag)
			if err != nil {
				return err
			}
			apks := append([]string{prep.Base}, prep.Splits...)
			ks := workspace.DebugKeystore(cfg.cwd)
			for i, p := range apks {
				signed, err := cfg.apk.Sign(ctx, p, ks)
				if err != nil {
					return err
				}
				if signed.APK != "" {
					apks[i] = signed.APK
				}
			}
			if err := cfg.apk.Install(ctx, dev.Serial, apks...); err != nil {
				return err
			}
			if asJSON {
				return writeInstallJSON(cmd.OutOrStdout(), apks[0], apks[1:], dev.Serial)
			}
			return writeInstallHuman(cmd.OutOrStdout(), apks, prep.Encoded)
		},
	}
	cmd.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	cmd.Flags().String("package", "", "package name (required when installing a decode dir outside .jdt/<pkg>/decode)")
	return cmd
}

func prepareInstall(ctx context.Context, cfg runConfig, target, pkgFlag string) (installPrep, error) {
	st, err := os.Stat(target)
	if err == nil && !st.IsDir() {
		return installPrep{Base: target}, nil
	}
	pkg := pkgFlag
	decodeDir := ""
	if err == nil && st.IsDir() {
		if !isDecodedTree(target) {
			return installPrep{}, fmt.Errorf("%w: not an apktool decode directory", apk.ErrUsage)
		}
		if pkg == "" {
			pkg, err = packageFromDecodeDir(cfg.cwd, target)
			if err != nil {
				return installPrep{}, err
			}
		}
		decodeDir = target
	} else {
		pkg = target
		layout, err := workspace.ForPackage(cfg.cwd, pkg)
		if err != nil {
			return installPrep{}, err
		}
		if !isDecodedTree(layout.Decode) {
			return installPrep{}, fmt.Errorf("install: decode: %w", os.ErrNotExist)
		}
		decodeDir = layout.Decode
	}
	base, encoded, err := buildPatched(ctx, cfg, decodeDir, pkg)
	if err != nil {
		return installPrep{}, err
	}
	splits, err := stageSplits(cfg.cwd, pkg)
	if err != nil {
		return installPrep{}, err
	}
	return installPrep{Base: base, Splits: splits, Encoded: encoded}, nil
}

func stageSplits(cwd, pkg string) ([]string, error) {
	layout, err := workspace.ForPackage(cwd, pkg)
	if err != nil {
		return nil, err
	}
	src, err := apk.ListSplits(layout.APK)
	if err != nil {
		return nil, err
	}
	dests := make([]string, 0, len(src))
	for _, s := range src {
		dst := filepath.Join(layout.Patched, filepath.Base(s))
		if err := copyFile(s, dst); err != nil {
			return nil, err
		}
		dests = append(dests, dst)
	}
	return dests, nil
}

func copyFile(src, dst string) error {
	if src == dst {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		_ = in.Close()
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		_ = in.Close()
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeOut := out.Close()
	closeIn := in.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeOut != nil {
		return closeOut
	}
	return closeIn
}

func buildPatched(ctx context.Context, cfg runConfig, decodedDir, pkg string) (string, bool, error) {
	layout, err := workspace.ForPackage(cfg.cwd, pkg)
	if err != nil {
		return "", false, err
	}
	if err := layout.Ensure(); err != nil {
		return "", false, err
	}
	out := filepath.Join(layout.Patched, "base.apk")
	art, err := cfg.apk.Build(ctx, decodedDir, out)
	if err != nil {
		return "", false, err
	}
	if art.APK != "" {
		out = art.APK
	}
	return out, true, nil
}

func isDecodedTree(dir string) bool {
	for _, name := range []string{"apktool.yml", "AndroidManifest.xml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func packageFromDecodeDir(cwd, dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(filepath.Join(absCwd, ".jdt"), absDir)
	if err != nil {
		return "", fmt.Errorf("%w: --package required for decode dir", apk.ErrUsage)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("%w: --package required for decode dir", apk.ErrUsage)
	}
	parts := strings.Split(rel, "/")
	if len(parts) != 2 || parts[1] != "decode" {
		return "", fmt.Errorf("%w: --package required for decode dir", apk.ErrUsage)
	}
	return parts[0], nil
}

func writeInstallHuman(w io.Writer, apks []string, encoded bool) error {
	if len(apks) == 0 {
		return nil
	}
	if encoded {
		if _, err := fmt.Fprintf(w, "[ok] encode: %s\n", apks[0]); err != nil {
			return err
		}
	}
	for _, p := range apks {
		if _, err := fmt.Fprintf(w, "[ok] sign: %s\n", p); err != nil {
			return err
		}
	}
	for _, p := range apks {
		if _, err := fmt.Fprintf(w, "[ok] install: %s\n", p); err != nil {
			return err
		}
	}
	return nil
}

type installJSON struct {
	APK    string   `json:"apk"`
	Splits []string `json:"splits"`
	Serial string   `json:"serial"`
}

func writeInstallJSON(w io.Writer, apkPath string, splits []string, serial string) error {
	if splits == nil {
		splits = []string{}
	}
	b, err := json.Marshal(installJSON{APK: apkPath, Splits: splits, Serial: serial})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
