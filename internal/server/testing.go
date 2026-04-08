package server

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

// ShouldVaryAcceptForTest exposes vary-header behavior for external tests.
func ShouldVaryAcceptForTest(sourceMediaType, explicitFormat string) bool {
	return shouldVaryAccept(sourceMediaType, explicitFormat)
}

// MetricsMiddlewareForTest exposes HTTP metrics middleware for external tests.
func MetricsMiddlewareForTest(obs observabilityx.Observability) fiber.Handler {
	return metricsMiddleware(obs)
}

// SetAssetDeliveryForTest exposes delivery tagging for external tests.
func SetAssetDeliveryForTest(c fiber.Ctx, delivery string) {
	setAssetDelivery(c, delivery)
}

// PublishVariantServedForTest exposes variant-served event publishing for external tests.
func PublishVariantServedForTest(
	ctx context.Context,
	result *resolver.Result,
	bus eventx.BusRuntime,
	logger *slog.Logger,
) {
	publishVariantServed(ctx, result, bus, logger)
}

// NewAppForTest exposes server construction for external tests.
func NewAppForTest(
	cfg *config.Config,
	logger *slog.Logger,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	bus eventx.BusRuntime,
) *fiber.App {
	healthChecks := newHealthCheckDefinitions(cfg, cat)
	app, err := newServerFromDeps(cfg, newServerRegistrations(
		newMiddlewareRegistration(cfg, logger, nil),
		newHealthRoutesRegistration(cat, healthChecks),
		newRobotsRouteRegistration(cfg, logger, cat, bodyCache),
		newAssetRouteRegistration(cfg, logger, assetResolver, pipelineSvc, bodyCache, bus),
	))
	if err != nil {
		panic(err)
	}
	return app
}

// NewHealthModuleForTest exposes the dix health-check setup for external tests.
func NewHealthModuleForTest(cfg *config.Config, cat catalog.Catalog) dix.Module {
	return dix.NewModule("server_health_test",
		dix.WithModuleProviders(
			dix.Provider0(func() *config.Config { return cfg }),
			dix.Provider0(func() catalog.Catalog { return cat }),
			dix.Provider2(newHealthCheckDefinitions),
		),
		dix.WithModuleSetups(
			dix.Setup(registerHealthCheckSetup),
		),
	)
}
