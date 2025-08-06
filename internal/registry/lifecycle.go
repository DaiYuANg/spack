package registry

import (
	"io/fs"
	"path/filepath"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type CollectDependency struct {
	fx.In
	Cfg      *config.Config
	Registry Registry
	Logger   *zap.SugaredLogger
	Pool     *ants.Pool
}

func collect(dependency CollectDependency) error {
	root := dependency.Cfg.Spa.Static
	logger := dependency.Logger
	registry := dependency.Registry
	pool := dependency.Pool
	if root == "" {
		logger.Warnf("No static file")
		return nil
	}
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		err = pool.Submit(func() {
			hash, err := pkg.FileHash(path)
			if err != nil {
				logger.Errorf("hash err%e", err)
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
				logger.Errorf("register err%e", err)
			}
		})
		if err != nil {
			return err
		}

		return nil
	})
}
