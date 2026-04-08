// Package cmd wires the CLI and application container.
package cmd

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/runtime"
	"github.com/daiyuang/spack/internal/task"
	"github.com/daiyuang/spack/internal/validation"
)

func createContainer(loadOptions config.LoadOptions, userModules ...dix.Module) (*dix.App, error) {
	allModules := collectionx.NewListWithCapacity[dix.Module](5 + len(userModules))
	allModules.Add(validation.Module,
		config.NewModule(loadOptions),
		logger.Module,
		catalog.Module,
		runtime.Module,
		task.Module,
	)
	allModules.Add(userModules...)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("could not read build info")
	}
	instance := dix.New(
		"spack",
		dix.WithVersion(info.Main.Version),
		dix.WithModules(allModules.Values()...),
		dix.WithLoggerFrom0(func() *slog.Logger {
			return logger.Bootstrap(loadOptions)
		}),
	)
	err := instance.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate dix container: %w", err)
	}
	return instance, nil
}
