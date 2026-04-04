// Package pkg contains small shared helpers used across packages.
package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func FileHash(path string, includeExt bool) (string, error) {
	// #nosec G304 -- callers pass local filesystem paths intentionally.
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hashing: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			return
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("copy file into hasher: %w", err)
	}

	if includeExt {
		ext := filepath.Ext(path)
		if _, err := hasher.Write([]byte(ext)); err != nil {
			return "", fmt.Errorf("write file extension into hasher: %w", err)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func FileHashOnly(path string) (string, error) {
	return FileHash(path, false)
}

func FileHashWithExt(path string) (string, error) {
	return FileHash(path, true)
}
