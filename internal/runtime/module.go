// Package runtime wires startup lifecycle and long-running services.
package runtime

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
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
	cfg, err := dix.ResolveAs[*config.Config](c)
	if err != nil {
		return err
	}
	src, err := dix.ResolveAs[source.Source](c)
	if err != nil {
		return err
	}
	cat, err := dix.ResolveAs[catalog.Catalog](c)
	if err != nil {
		return err
	}
	pipelineSvc, err := dix.ResolveAs[*pipeline.Service](c)
	if err != nil {
		return err
	}
	pipelineMetrics, err := dix.ResolveAs[*pipeline.Metrics](c)
	if err != nil {
		return err
	}
	logger, err := dix.ResolveAs[*slog.Logger](c)
	if err != nil {
		return err
	}
	app, err := dix.ResolveAs[*fiber.App](c)
	if err != nil {
		return err
	}
	metricsAdapter, err := dix.ResolveAs[*obsprom.Adapter](c)
	if err != nil {
		return err
	}

	logConfigLifecycle(lc, cfg, logger)
	bootstrapCatalog(lc, cfg, src, cat, pipelineSvc, logger)
	httpLifecycle(lc, app, cfg, cat, logger)
	debugLifecycle(lc, cfg, logger, pipelineMetrics, metricsAdapter)
	return nil
}
