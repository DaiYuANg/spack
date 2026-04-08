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
	"github.com/daiyuang/spack/internal/source"
	"github.com/gofiber/fiber/v3"
)

var Module = dix.NewModule("runtime",
	dix.WithModuleProviders(
		dix.Provider6(newCatalogBootstrapRuntime),
		dix.Provider4(newHTTPRuntime),
		dix.Provider4(newDebugRuntime),
		dix.Provider3(newRuntimeState),
	),
	dix.WithModuleHooks(
		dix.OnStart(startRuntime),
		dix.OnStop(stopRuntime),
	),
)

type catalogBootstrapRuntime struct {
	cfg         *config.Config
	src         source.Source
	cat         catalog.Catalog
	bodyCache   *assetcache.Cache
	pipelineSvc *pipeline.Service
	logger      *slog.Logger
}

func newCatalogBootstrapRuntime(
	cfg *config.Config,
	src source.Source,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	pipelineSvc *pipeline.Service,
	logger *slog.Logger,
) catalogBootstrapRuntime {
	return catalogBootstrapRuntime{
		cfg:         cfg,
		src:         src,
		cat:         cat,
		bodyCache:   bodyCache,
		pipelineSvc: pipelineSvc,
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

type runtimeState struct {
	bootstrap catalogBootstrapRuntime
	http      httpRuntime
	debug     *debugRuntime
}

func newRuntimeState(
	bootstrap catalogBootstrapRuntime,
	http httpRuntime,
	debug *debugRuntime,
) *runtimeState {
	return &runtimeState{
		bootstrap: bootstrap,
		http:      http,
		debug:     debug,
	}
}

func newDebugRuntime(
	cfg *config.Config,
	logger *slog.Logger,
	pipelineMetrics *pipeline.Metrics,
	metricsAdapter *obsprom.Adapter,
) *debugRuntime {
	return buildDebugRuntime(cfg, logger, pipelineMetrics, metricsAdapter)
}
