// Package task for schedule task
package task

import (
	"cmp"
	"context"
	"log/slog"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

var Module = dix.NewModule("task",
	dix.WithModuleProviders(
		dix.ProviderErr1(newScheduler),
		dix.Provider4(newSourceRescanRuntime),
		dix.Provider4(newArtifactJanitorRuntime),
		dix.Provider4(newCacheWarmerRuntime),
		dix.Provider1(newSourceRescanTaskRegistration),
		dix.Provider1(newArtifactJanitorTaskRegistration),
		dix.Provider1(newCacheWarmerTaskRegistration),
		dix.Provider3(newTaskRegistrations),
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
		return nil, oops.In("task").Owner("scheduler").Wrap(err)
	}
	return scheduler, nil
}

type taskRegistration struct {
	Order    int
	Name     string
	Register func(context.Context, gocron.Scheduler) (bool, error)
}

type sourceRescanTaskRegistration struct {
	taskRegistration
}

type artifactJanitorTaskRegistration struct {
	taskRegistration
}

type cacheWarmerTaskRegistration struct {
	taskRegistration
}

func newTaskRegistration(
	order int,
	name string,
	register func(context.Context, gocron.Scheduler) (bool, error),
) taskRegistration {
	return taskRegistration{
		Order:    order,
		Name:     strings.TrimSpace(name),
		Register: register,
	}
}

func newTaskRegistrations(
	sourceRescan sourceRescanTaskRegistration,
	artifactJanitor artifactJanitorTaskRegistration,
	cacheWarmer cacheWarmerTaskRegistration,
) collectionx.List[taskRegistration] {
	return collectionx.NewList(
		sourceRescan.taskRegistration,
		artifactJanitor.taskRegistration,
		cacheWarmer.taskRegistration,
	).Sort(func(left, right taskRegistration) int {
		if left.Order != right.Order {
			return cmp.Compare(left.Order, right.Order)
		}
		return cmp.Compare(left.Name, right.Name)
	})
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

func newSourceRescanTaskRegistration(runtime *sourceRescanRuntime) sourceRescanTaskRegistration {
	return sourceRescanTaskRegistration{newTaskRegistration(100, "source_rescan", func(ctx context.Context, scheduler gocron.Scheduler) (bool, error) {
		return registerSourceRescanTask(ctx, scheduler, runtime)
	})}
}

type artifactJanitorRuntime struct {
	store     artifact.Store
	catalog   catalog.Catalog
	bodyCache *assetcache.Cache
	logger    *slog.Logger
}

func newArtifactJanitorRuntime(
	store artifact.Store,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	logger *slog.Logger,
) *artifactJanitorRuntime {
	return &artifactJanitorRuntime{
		store:     store,
		catalog:   cat,
		bodyCache: bodyCache,
		logger:    logger,
	}
}

func newArtifactJanitorTaskRegistration(runtime *artifactJanitorRuntime) artifactJanitorTaskRegistration {
	return artifactJanitorTaskRegistration{newTaskRegistration(200, "artifact_janitor", func(ctx context.Context, scheduler gocron.Scheduler) (bool, error) {
		return registerArtifactJanitorTask(ctx, scheduler, runtime)
	})}
}

type cacheWarmerRuntime struct {
	cfg       *config.Config
	catalog   catalog.Catalog
	bodyCache *assetcache.Cache
	logger    *slog.Logger
}

func newCacheWarmerRuntime(
	cfg *config.Config,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	logger *slog.Logger,
) *cacheWarmerRuntime {
	return &cacheWarmerRuntime{
		cfg:       cfg,
		catalog:   cat,
		bodyCache: bodyCache,
		logger:    logger,
	}
}

func newCacheWarmerTaskRegistration(runtime *cacheWarmerRuntime) cacheWarmerTaskRegistration {
	return cacheWarmerTaskRegistration{newTaskRegistration(300, "cache_warmer", func(ctx context.Context, scheduler gocron.Scheduler) (bool, error) {
		return registerCacheWarmerTask(ctx, scheduler, runtime)
	})}
}

func startScheduledTasks(
	ctx context.Context,
	scheduler gocron.Scheduler,
	registrations collectionx.List[taskRegistration],
) error {
	registered := lo.FilterMap(registrations.Values(), func(registration taskRegistration, _ int) (taskRegistration, bool) {
		if registration.Register == nil {
			return taskRegistration{}, false
		}
		return registration, true
	})
	if len(registered) == 0 {
		return nil
	}

	started := false
	for _, registration := range registered {
		enabled, err := registration.Register(ctx, scheduler)
		if err != nil {
			return oops.In("task").Owner(registration.Name).Wrap(err)
		}
		if enabled {
			started = true
		}
	}
	if started {
		scheduler.Start()
	}
	return nil
}
