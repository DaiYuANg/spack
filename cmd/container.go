// Package cmd wires the CLI and application container.
package cmd

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/runtime"
	"github.com/daiyuang/spack/internal/validation"
)

func createContainer(loadOptions config.LoadOptions, userModules ...dix.Module) *dix.App {
	allModules := collectionx.NewListWithCapacity[dix.Module](5 + len(userModules))
	allModules.Add(validation.Module,
		config.NewModule(loadOptions),
		logger.Module,
		catalog.Module,
		runtime.Module,
	)
	allModules.Add(userModules...)
	return dix.New(
		"spack",
		dix.WithModules(allModules.Values()...),
		dix.WithLoggerFrom0(func() *slog.Logger {
			return logger.Bootstrap(loadOptions)
		}),
	)
}
