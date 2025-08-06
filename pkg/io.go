package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func FileHash(path string, includeExt bool) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close() // 忽略错误避免 panic
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	if includeExt {
		ext := filepath.Ext(path)
		hasher.Write([]byte(ext))
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
func FileHashOnly(path string) (string, error) {
	return FileHash(path, false)
}

func FileHashWithExt(path string) (string, error) {
	return FileHash(path, true)
}
