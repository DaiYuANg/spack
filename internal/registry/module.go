package registry

import (
	"context"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/pkg"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"io/fs"
	"path/filepath"
)

var Module = fx.Module("registry", fx.Provide(
	fx.Annotate(
		NewInMemoryRegistry,
		fx.As(new(Registry)),
	),
	newContext), fx.Invoke(collect))

func newContext() context.Context {
	return context.Background()
}

func collect(config *config.Config, registry Registry, logger *zap.SugaredLogger) error {
	root := config.Spa.Static
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		hash, err := pkg.FileHash(path)
		if err != nil {
			return err
		}
		entry := &Entry{
			ActualPath:  path,
			MimeType:    pkg.DetectMIME(path),
			RequestPath: filepath.ToSlash(path[len(root):]),
			Version:     hash,
		}
		logger.Debugf("register %v", entry)
		err = registry.Register(path, entry)
		if err != nil {
			return err
		}
		return nil
	})
}
