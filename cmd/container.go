// Package cmd wires the CLI and application container.
package cmd

import (
	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
	"github.com/daiyuang/spack/internal/appmeta"
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
	allModules := collectionx.NewListWithCapacity[dix.Module](8 + len(userModules))
	allModules.Add(appmeta.Module,
		validation.Module,
		config.NewModule(loadOptions),
		spacklogger.Module,
		metrics.Module,
		catalog.Module,
		runtime.Module,
		task.Module,
	)
	allModules.Add(userModules...)
	instance := dix.New(
		"spack",
		dix.WithModules(allModules.Values()...),
		dix.WithRunStopTimeout(dix.DefaultRunStopTimeout),
	)
	err := instance.Validate()
	if err != nil {
		return nil, oops.In("command").Owner("container").Wrap(err)
	}
	return instance, nil
}
