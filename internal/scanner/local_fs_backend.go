package scanner

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// localFSBackend 实现 Backend
type localFSBackend struct {
	root   string
	logger *slog.Logger
}

// NewLocalFSBackend 构建 LocalFS backend
func NewLocalFSBackend(root string, logger *slog.Logger) Backend {
	logger.Info("Local fs backend root%s", root)
	return &localFSBackend{root: root, logger: logger}
}

func (b *localFSBackend) Walk(walkFn func(obj *ObjectInfo) error) error {
	return filepath.Walk(b.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(b.root, path)
		if err != nil {
			return err
		}
		// 标准化分隔符
		key := filepath.ToSlash(rel)

		obj := &ObjectInfo{
			Key:   key,
			Size:  info.Size(),
			IsDir: info.IsDir(),
			Reader: func() (io.ReadCloser, error) {
				if info.IsDir() {
					return nil, nil
				}
				return os.Open(path)
			},
		}

		return walkFn(obj)
	})
}

func (b *localFSBackend) Stat(key string) (*ObjectInfo, error) {
	full := filepath.Join(b.root, filepath.FromSlash(key))
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}

	return &ObjectInfo{
		Key:   key,
		Size:  info.Size(),
		IsDir: info.IsDir(),
		Reader: func() (io.ReadCloser, error) {
			if info.IsDir() {
				return nil, nil
			}
			return os.Open(full)
		},
	}, nil
}
