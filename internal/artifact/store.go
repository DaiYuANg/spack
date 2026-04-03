package artifact

import (
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmpPath := path + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}
