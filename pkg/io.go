// Package pkg contains small shared helpers used across packages.
package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func HashFile(path string) (string, error) {
	// #nosec G304 -- paths come from the scanned local asset tree.
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hashing: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			return
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("copy file into hasher: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
