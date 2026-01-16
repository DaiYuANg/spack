package scanner

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/daiyuang/spack/internal/model"
	"github.com/samber/oops"
)

// localFSBackend 实现 Backend
type localFSBackend struct {
	root   string
	logger *slog.Logger
}

// NewLocalFSBackend 构建 LocalFS backend
func NewLocalFSBackend(root string, logger *slog.Logger) Backend {
	logger.Info("Local fs backend root", slog.String("root", root))
	return &localFSBackend{root: root, logger: logger}
}

func (b *localFSBackend) Walk(walkFn func(obj *model.ObjectInfo) error) error {
	return filepath.Walk(b.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return oops.Wrap(err)
		}
		obj, err := model.NewObjectInfo(b.root, path, info)
		if err != nil {
			return err
		}
		return walkFn(obj)
	})
}

func (b *localFSBackend) Stat(key string) (*model.ObjectInfo, error) {
	full := filepath.Join(b.root, filepath.FromSlash(key))
	info, err := os.Stat(full)
	if err != nil {
		return nil, oops.Wrap(err)
	}
	return model.NewObjectInfo(b.root, full, info)
}
