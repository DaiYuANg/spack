package artifact

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Store interface {
	Root() string
	PathFor(assetPath, sourceHash, namespace, suffix string) string
	Write(path string, data []byte) error
}

type LocalStore struct {
	root string
}

func newLocalStore(root string) Store {
	return &LocalStore{root: root}
}

func (s *LocalStore) Root() string {
	return s.root
}

func (s *LocalStore) PathFor(assetPath, sourceHash, namespace, suffix string) string {
	cleanPath := filepath.FromSlash(strings.TrimPrefix(assetPath, "/"))
	cleanPath = filepath.Clean(cleanPath)
	if cleanPath == "." {
		cleanPath = "index"
	}

	return filepath.Join(s.root, namespace, sourceHash, cleanPath+suffix)
}

func (s *LocalStore) Write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	tmpPath := path + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write artifact temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return errors.Join(fmt.Errorf("rename artifact temp file: %w", err), fmt.Errorf("cleanup artifact temp file: %w", removeErr))
		}
		return fmt.Errorf("rename artifact temp file: %w", err)
	}

	return nil
}
