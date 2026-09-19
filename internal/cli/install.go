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
			apkPath, encoded, err := prepareInstall(ctx, cfg, args[0], pkgFlag)
			if err != nil {
				return err
			}
			signed, err := cfg.apk.Sign(ctx, apkPath, workspace.DebugKeystore(cfg.cwd))
			if err != nil {
				return err
			}
			if signed.APK != "" {
				apkPath = signed.APK
			}
			if err := cfg.apk.Install(ctx, dev.Serial, apkPath); err != nil {
				return err
			}
			if asJSON {
				return writeInstallJSON(cmd.OutOrStdout(), apkPath, dev.Serial)
			}
			return writeInstallHuman(cmd.OutOrStdout(), apkPath, encoded)
		},
	}
	cmd.Flags().StringP("serial", "s", "", "adb serial (required if multiple devices)")
	cmd.Flags().String("package", "", "package name (required when installing a decode dir outside .jdt/<pkg>/decode)")
	return cmd
}

func prepareInstall(ctx context.Context, cfg runConfig, target, pkgFlag string) (string, bool, error) {
	st, err := os.Stat(target)
	if err == nil && !st.IsDir() {
		return target, false, nil
	}
	if err == nil && st.IsDir() {
		if !isDecodedTree(target) {
			return "", false, fmt.Errorf("%w: not an apktool decode directory", apk.ErrUsage)
		}
		pkg := pkgFlag
		if pkg == "" {
			pkg, err = packageFromDecodeDir(cfg.cwd, target)
			if err != nil {
				return "", false, err
			}
		}
		return buildPatched(ctx, cfg, target, pkg)
	}
	layout, err := workspace.ForPackage(cfg.cwd, target)
	if err != nil {
		return "", false, err
	}
	if !isDecodedTree(layout.Decode) {
		return "", false, fmt.Errorf("install: decode: %w", os.ErrNotExist)
	}
	return buildPatched(ctx, cfg, layout.Decode, target)
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

func writeInstallHuman(w io.Writer, apkPath string, encoded bool) error {
	if encoded {
		if _, err := fmt.Fprintf(w, "[ok] encode: %s\n", apkPath); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "[ok] sign: %s\n", apkPath); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "[ok] install: %s\n", apkPath)
	return err
}

type installJSON struct {
	APK    string `json:"apk"`
	Serial string `json:"serial"`
}

func writeInstallJSON(w io.Writer, apkPath, serial string) error {
	b, err := json.Marshal(installJSON{APK: apkPath, Serial: serial})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
