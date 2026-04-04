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
	dix.WithModuleSetups(
		dix.SetupWithMetadata(setupRuntime, dix.SetupMetadata{
			Label: "SetupRuntime",
			Dependencies: dix.ServiceRefs(
				dix.TypedService[*config.Config](),
				dix.TypedService[source.Source](),
				dix.TypedService[catalog.Catalog](),
				dix.TypedService[*assetcache.Cache](),
				dix.TypedService[*pipeline.Service](),
				dix.TypedService[*pipeline.Metrics](),
				dix.TypedService[*slog.Logger](),
				dix.TypedService[*fiber.App](),
				dix.TypedService[*obsprom.Adapter](),
			),
		}),
	),
)

func setupRuntime(c *dix.Container, lc dix.Lifecycle) error {
	cfg := dix.MustResolveAs[*config.Config](c)
	src := dix.MustResolveAs[source.Source](c)
	cat := dix.MustResolveAs[catalog.Catalog](c)
	bodyCache := dix.MustResolveAs[*assetcache.Cache](c)
	pipelineSvc := dix.MustResolveAs[*pipeline.Service](c)
	pipelineMetrics := dix.MustResolveAs[*pipeline.Metrics](c)
	logger := dix.MustResolveAs[*slog.Logger](c)
	app := dix.MustResolveAs[*fiber.App](c)
	metricsAdapter := dix.MustResolveAs[*obsprom.Adapter](c)

	logConfigLifecycle(lc, cfg, logger)
	bootstrapCatalog(lc, cfg, src, cat, bodyCache, pipelineSvc, logger)
	httpLifecycle(lc, app, cfg, cat, logger)
	debugLifecycle(lc, cfg, logger, pipelineMetrics, metricsAdapter)
	return nil
}
