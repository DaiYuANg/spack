// Package task for schedule task
package task

import (
	"cmp"
	"context"
	cxlist "github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/observabilityx"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/sourcecatalog"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/oops"
	"log/slog"
	"strings"
	"sync"
)

var Module = dix.NewModule("task",
	dix.WithModuleProviders(
		dix.Provider0(NewRuntimeMetrics),
		dix.ProviderErr2(newScheduler),
		dix.Provider6(newSourceRescanRuntime),
		dix.Provider6(newArtifactJanitorRuntime),
		dix.Provider5(newCacheWarmerRuntime),
		dix.Provider1(newSourceRescanWatcher),
		dix.Provider1(newSourceRescanTaskRegistration),
		dix.Provider1(newArtifactJanitorTaskRegistration),
		dix.Provider1(newCacheWarmerTaskRegistration),
		dix.Provider3(newTaskRegistrations),
	),
	dix.WithModuleHooks(
		dix.OnStart2(startTaskRuntime),
		dix.OnStart(startSourceRescanWatcher),
		dix.OnStop(stopSourceRescanWatcher),
		dix.OnStop(stopTaskRuntime),
	),
)

func newScheduler(logger *slog.Logger, runtimeMetrics *RuntimeMetrics) (gocron.Scheduler, error) {
	scheduler, err := gocron.NewScheduler(
		gocron.WithLogger(logger),
		gocron.WithSchedulerMonitor(runtimeMetrics),
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
) *cxlist.List[taskRegistration] {
	return cxlist.NewList[taskRegistration](
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
	scanner    sourcecatalog.Scanner
	catalog    catalog.Catalog
	catMetrics *catalog.RuntimeMetrics
	bodyCache  *assetcache.Cache
	logger     *slog.Logger
	obs        observabilityx.Observability
	rescanMu   sync.Mutex
}

func newSourceRescanRuntime(
	scanner sourcecatalog.Scanner,
	cat catalog.Catalog,
	catMetrics *catalog.RuntimeMetrics,
	bodyCache *assetcache.Cache,
	logger *slog.Logger,
	obs observabilityx.Observability,
) *sourceRescanRuntime {
	return &sourceRescanRuntime{
		scanner:    scanner,
		catalog:    cat,
		catMetrics: catMetrics,
		bodyCache:  bodyCache,
		logger:     logger,
		obs:        observabilityx.Normalize(obs, logger),
	}
}

func newSourceRescanTaskRegistration(runtime *sourceRescanRuntime) sourceRescanTaskRegistration {
	return sourceRescanTaskRegistration{newTaskRegistration(100, "source_rescan", func(ctx context.Context, scheduler gocron.Scheduler) (bool, error) {
		return registerSourceRescanTask(ctx, scheduler, runtime)
	})}
}

type sourceRescanWatcher struct {
	runtime *sourceRescanRuntime
	cancel  context.CancelFunc
	done    chan struct{}
}

func newSourceRescanWatcher(runtime *sourceRescanRuntime) *sourceRescanWatcher {
	return &sourceRescanWatcher{runtime: runtime}
}

type artifactJanitorRuntime struct {
	store      artifact.Store
	catalog    catalog.Catalog
	catMetrics *catalog.RuntimeMetrics
	bodyCache  *assetcache.Cache
	logger     *slog.Logger
	obs        observabilityx.Observability
}

func newArtifactJanitorRuntime(
	store artifact.Store,
	cat catalog.Catalog,
	catMetrics *catalog.RuntimeMetrics,
	bodyCache *assetcache.Cache,
	logger *slog.Logger,
	obs observabilityx.Observability,
) *artifactJanitorRuntime {
	return &artifactJanitorRuntime{
		store:      store,
		catalog:    cat,
		catMetrics: catMetrics,
		bodyCache:  bodyCache,
		logger:     logger,
		obs:        observabilityx.Normalize(obs, logger),
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
	obs       observabilityx.Observability
}

func newCacheWarmerRuntime(
	cfg *config.Config,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	logger *slog.Logger,
	obs observabilityx.Observability,
) *cacheWarmerRuntime {
	return &cacheWarmerRuntime{
		cfg:       cfg,
		catalog:   cat,
		bodyCache: bodyCache,
		logger:    logger,
		obs:       observabilityx.Normalize(obs, logger),
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
	registrations *cxlist.List[taskRegistration],
) error {
	registered := cxlist.FlatMapList[taskRegistration, taskRegistration](registrations, func(_ int, registration taskRegistration) []taskRegistration {
		if registration.Register == nil {
			return nil
		}
		return []taskRegistration{registration}
	})
	if registered.IsEmpty() {
		return nil
	}

	started := false
	var registerErr error
	registered.Range(func(_ int, registration taskRegistration) bool {
		enabled, err := registration.Register(ctx, scheduler)
		if err != nil {
			registerErr = oops.In("task").Owner(registration.Name).Wrap(err)
			return false
		}
		if enabled {
			started = true
		}
		return true
	})
	if registerErr != nil {
		return registerErr
	}
	if started {
		scheduler.Start()
	}
	return nil
}
