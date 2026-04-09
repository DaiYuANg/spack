package pipeline

import (
	"context"

	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/panjf2000/ants/v2"
)

var Module = dix.NewModule("pipeline",
	dix.WithModuleProviders(
		dix.Provider0(newMetrics),
		dix.Provider0(newImageEngine),
		dix.Provider4(newImageStage),
		dix.Provider4(newCompressionStage),
		dix.Provider2(newStageRegistrations),
		dix.Provider1(newStages),
		dix.Provider6(newServiceDeps),
		dix.Provider4(newService),
	),
	dix.WithModuleHooks(
		dix.OnStart(startServiceLifecycle),
		dix.OnStop(func(ctx context.Context, svc *Service) error {
			return svc.stop(ctx)
		}),
	),
)

func newStageRegistrations(image *imageStage, compression *compressionStage) collectionx.List[stageRegistration] {
	return collectionx.NewList(
		newStageRegistration(100, image),
		newStageRegistration(200, compression),
	)
}

func newStages(registrations collectionx.List[stageRegistration]) collectionx.List[Stage] {
	return buildStages(registrations)
}

type serviceDeps struct {
	metrics    *Metrics
	stages     collectionx.List[Stage]
	bus        eventx.BusRuntime
	pool       *ants.Pool
	obs        observabilityx.Observability
	catMetrics *catalog.RuntimeMetrics
}

func newServiceDeps(
	metrics *Metrics,
	stages collectionx.List[Stage],
	bus eventx.BusRuntime,
	pool *ants.Pool,
	obs observabilityx.Observability,
	catMetrics *catalog.RuntimeMetrics,
) serviceDeps {
	return serviceDeps{
		metrics:    metrics,
		stages:     stages,
		bus:        bus,
		pool:       pool,
		obs:        observabilityx.Normalize(obs, nil),
		catMetrics: catMetrics,
	}
}

func newService(
	cfg *config.Compression,
	logger *slog.Logger,
	cat catalog.Catalog,
	deps serviceDeps,
) *Service {
	workers := max(cfg.Workers, 1)
	queueSize := resolveQueueSize(cfg, workers)

	svc := newServiceState(cfg, logger, cat, deps.metrics, deps.stages, deps.bus, deps.pool, deps.obs, deps.catMetrics, queueSize)
	svc.initializeMetrics(queueSize)
	return svc
}

func startServiceLifecycle(ctx context.Context, svc *Service) error {
	workers := max(svc.cfg.Workers, 1)
	return svc.start(ctx, workers, cap(svc.tasks))
}
