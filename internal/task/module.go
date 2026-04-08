// Package task for schedule task
package task

import (
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/source"
	"github.com/go-co-op/gocron/v2"
)

var Module = dix.NewModule("task",
	dix.WithModuleProviders(
		dix.ProviderErr1(newScheduler),
		dix.Provider4(newSourceRescanRuntime),
	),
	dix.WithModuleHooks(
		dix.OnStart2(startTaskRuntime),
		dix.OnStop(stopTaskRuntime),
	),
)

func newScheduler(logger *slog.Logger) (gocron.Scheduler, error) {
	scheduler, err := gocron.NewScheduler(
		gocron.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}
	return scheduler, nil
}

type sourceRescanRuntime struct {
	src       source.Source
	catalog   catalog.Catalog
	bodyCache *assetcache.Cache
	logger    *slog.Logger
}

func newSourceRescanRuntime(
	src source.Source,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	logger *slog.Logger,
) *sourceRescanRuntime {
	return &sourceRescanRuntime{
		src:       src,
		catalog:   cat,
		bodyCache: bodyCache,
		logger:    logger,
	}
}
