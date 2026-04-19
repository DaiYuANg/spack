// Package cmd wires the CLI and application container.
package cmd

import (
	"runtime/debug"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	spacklogger "github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/metrics"
	"github.com/daiyuang/spack/internal/runtime"
	"github.com/daiyuang/spack/internal/task"
	"github.com/daiyuang/spack/internal/validation"
	"github.com/samber/oops"
)

func createContainer(loadOptions config.LoadOptions, userModules ...dix.Module) (*dix.App, error) {
	allModules := collectionx.NewListWithCapacity[dix.Module](7 + len(userModules))
	allModules.Add(validation.Module,
		config.NewModule(loadOptions),
		spacklogger.Module,
		metrics.Module,
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
	)
	err := instance.Validate()
	if err != nil {
		return nil, oops.In("command").Owner("container").Wrap(err)
	}
	return instance, nil
}
