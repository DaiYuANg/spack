// Package cmd wires the CLI and application container.
package cmd

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/runtime"
)

func createContainer(loadOptions config.LoadOptions, userModules ...dix.Module) *dix.App {
	allModules := make([]dix.Module, 0, 4+len(userModules))
	allModules = append(allModules,
		config.NewModule(loadOptions),
		logger.Module,
		catalog.Module,
	)
	allModules = append(allModules, userModules...)
	allModules = append(allModules, runtime.Module)

	return dix.New(
		"spack",
		dix.WithModules(allModules...),
		dix.WithLoggerFrom0(func() *slog.Logger {
			return logger.Bootstrap(loadOptions)
		}),
	)
}
