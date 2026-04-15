// Package runtime wires startup lifecycle and long-running services.
package runtime

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/server"
	"github.com/daiyuang/spack/internal/sourcecatalog"
	"github.com/daiyuang/spack/internal/task"
	"github.com/daiyuang/spack/internal/workerpool"
	"github.com/gofiber/fiber/v3"
)

var Module = dix.NewModule("runtime",
	dix.WithModuleProviders(
		dix.Provider5(newCatalogBootstrapDeps),
		dix.Provider3(newCatalogBootstrapRuntime),
		dix.Provider4(newHTTPRuntime),
		dix.Provider6(newDebugRuntimeDeps),
		dix.Provider3(newDebugRuntime),
	),
	dix.WithModuleHooks(
		dix.OnStart(logConfigOnStart),
		dix.OnStart(bootstrapCatalogOnStart),
		dix.OnStart2(startHTTPRuntime),
		dix.OnStart2(startDebugRuntime),
		dix.OnStop(stopDebugRuntime),
		dix.OnStop(stopHTTPRuntime),
	),
)

type catalogBootstrapDeps struct {
	scanner     sourcecatalog.Scanner
	cat         catalog.Catalog
	catMetrics  *catalog.RuntimeMetrics
	bodyCache   *assetcache.Cache
	pipelineSvc *pipeline.Service
}

func newCatalogBootstrapDeps(
	scanner sourcecatalog.Scanner,
	cat catalog.Catalog,
	catMetrics *catalog.RuntimeMetrics,
	bodyCache *assetcache.Cache,
	pipelineSvc *pipeline.Service,
) catalogBootstrapDeps {
	return catalogBootstrapDeps{
		scanner:     scanner,
		cat:         cat,
		catMetrics:  catMetrics,
		bodyCache:   bodyCache,
		pipelineSvc: pipelineSvc,
	}
}

type catalogBootstrapRuntime struct {
	cfg         *config.Config
	scanner     sourcecatalog.Scanner
	cat         catalog.Catalog
	catMetrics  *catalog.RuntimeMetrics
	bodyCache   *assetcache.Cache
	pipelineSvc *pipeline.Service
	logger      *slog.Logger
}

func newCatalogBootstrapRuntime(
	cfg *config.Config,
	deps catalogBootstrapDeps,
	logger *slog.Logger,
) catalogBootstrapRuntime {
	return catalogBootstrapRuntime{
		cfg:         cfg,
		scanner:     deps.scanner,
		cat:         deps.cat,
		catMetrics:  deps.catMetrics,
		bodyCache:   deps.bodyCache,
		pipelineSvc: deps.pipelineSvc,
		logger:      logger,
	}
}

type httpRuntime struct {
	app    *fiber.App
	cfg    *config.Config
	cat    catalog.Catalog
	logger *slog.Logger
}

func newHTTPRuntime(app *fiber.App, cfg *config.Config, cat catalog.Catalog, logger *slog.Logger) httpRuntime {
	return httpRuntime{
		app:    app,
		cfg:    cfg,
		cat:    cat,
		logger: logger,
	}
}

type debugRuntimeDeps struct {
	pipelineMetrics   *pipeline.Metrics
	catMetrics        *catalog.RuntimeMetrics
	serverMetrics     *server.RuntimeMetrics
	taskMetrics       *task.RuntimeMetrics
	workerpoolMetrics *workerpool.RuntimeMetrics
	metricsAdapter    *obsprom.Adapter
}

func newDebugRuntimeDeps(
	pipelineMetrics *pipeline.Metrics,
	catMetrics *catalog.RuntimeMetrics,
	serverMetrics *server.RuntimeMetrics,
	taskMetrics *task.RuntimeMetrics,
	workerpoolMetrics *workerpool.RuntimeMetrics,
	metricsAdapter *obsprom.Adapter,
) debugRuntimeDeps {
	return debugRuntimeDeps{
		pipelineMetrics:   pipelineMetrics,
		catMetrics:        catMetrics,
		serverMetrics:     serverMetrics,
		taskMetrics:       taskMetrics,
		workerpoolMetrics: workerpoolMetrics,
		metricsAdapter:    metricsAdapter,
	}
}

func newDebugRuntime(
	cfg *config.Config,
	logger *slog.Logger,
	deps debugRuntimeDeps,
) *debugRuntime {
	return buildDebugRuntime(cfg, logger, deps)
}
