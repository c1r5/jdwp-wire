package apk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (t *Tools) Sign(ctx context.Context, apkPath, keystore string) (Artifact, error) {
	if apkPath == "" || keystore == "" {
		return Artifact{}, fmt.Errorf("apk: sign: %w", ErrUsage)
	}
	look := t.look
	if look == nil {
		return Artifact{}, fmt.Errorf("apk: sign: %w: apksigner", ErrToolMissing)
	}
	if _, err := look("apksigner"); err == nil {
		if err := t.ensureKeystore(ctx, keystore); err != nil {
			return Artifact{}, err
		}
		res, err := t.r.Run(ctx, "apksigner", "sign",
			"--ks", keystore,
			"--ks-pass", "pass:android",
			"--ks-key-alias", "androiddebugkey",
			"--key-pass", "pass:android",
			"--in", apkPath,
			"--out", apkPath,
		)
		if err != nil {
			return Artifact{}, wrapRun("sign", "apksigner", res.Stderr, err)
		}
		return Artifact{APK: apkPath}, nil
	}
	if _, err := look("uber-apk-signer"); err == nil {
		res, err := t.r.Run(ctx, "uber-apk-signer", "-a", apkPath, "--allowResign", "--overwrite")
		if err != nil {
			return Artifact{}, wrapRun("sign", "uber-apk-signer", res.Stderr, err)
		}
		return Artifact{APK: apkPath}, nil
	}
	return Artifact{}, fmt.Errorf("apk: sign: %w: apksigner", ErrToolMissing)
}

func (t *Tools) ensureKeystore(ctx context.Context, keystore string) error {
	if _, err := os.Stat(keystore); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(keystore), 0o755); err != nil {
		return fmt.Errorf("apk: sign: %w", err)
	}
	res, err := t.r.Run(ctx, "keytool", "-genkeypair",
		"-keystore", keystore,
		"-storepass", "android",
		"-alias", "androiddebugkey",
		"-keypass", "android",
		"-keyalg", "RSA",
		"-keysize", "2048",
		"-validity", "10000",
		"-dname", "CN=Android Debug,O=Android,C=US",
		"-noprompt",
	)
	if err != nil {
		return wrapRun("sign", "keytool", res.Stderr, err)
	}
	return nil
}
