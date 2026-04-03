package source

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/daiyuang/spack/internal/config"
	"github.com/samber/oops"
)

type localFS struct {
	root   string
	logger *slog.Logger
}

func newLocalFS(cfg *config.Assets, logger *slog.Logger) (Source, error) {
	root := strings.TrimSpace(cfg.Root)
	if root == "" {
		return nil, fmt.Errorf("assets root is required")
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("assets root must be a directory: %s", root)
	}

	logger.Info("Source configured", slog.String("root", cfg.Root))
	return &localFS{
		root:   root,
		logger: logger,
	}, nil
}

func (s *localFS) Walk(walkFn func(File) error) error {
	return filepath.Walk(s.root, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return oops.Wrap(err)
		}

		rel, err := filepath.Rel(s.root, fullPath)
		if err != nil {
			return oops.Wrap(err)
		}

		return walkFn(File{
			Path:     filepath.ToSlash(rel),
			FullPath: fullPath,
			Size:     info.Size(),
			IsDir:    info.IsDir(),
			ModTime:  info.ModTime(),
		})
	})
}
