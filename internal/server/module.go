package server

import (
	"cmp"
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
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

var Module = dix.NewModule("server",
	dix.WithModuleProviders(
		dix.Provider0(NewRuntimeMetrics),
		dix.Provider4(newMiddlewareRegistration),
		dix.Provider2(newAssetRouteRuntime),
		dix.Provider2(newHealthCheckDefinitions),
		dix.Provider3(newHealthRoutesRegistration),
		dix.Provider4(newRobotsRouteRegistration),
		dix.Provider6(newAssetRouteRegistration),
		dix.Provider4(newServerRegistrations),
		dix.ProviderErr2(newServerFromDeps),
	),
	dix.WithModuleSetups(
		dix.Setup(registerHealthCheckSetup),
	),
)

type appRegistration struct {
	Order int
	Name  string
	Apply func(*fiber.App)
}

type middlewareRegistration struct {
	appRegistration
}

type healthRoutesRegistration struct {
	appRegistration
}

type robotsRouteRegistration struct {
	appRegistration
}

type assetRouteRegistration struct {
	appRegistration
}

type assetRouteRuntime struct {
	logger        *slog.Logger
	trackDelivery bool
}

func newAppRegistration(order int, name string, apply func(*fiber.App)) appRegistration {
	return appRegistration{
		Order: order,
		Name:  name,
		Apply: apply,
	}
}

func newMiddlewareRegistration(
	cfg *config.Config,
	logger *slog.Logger,
	obs observabilityx.Observability,
	runtimeMetrics *RuntimeMetrics,
) middlewareRegistration {
	return middlewareRegistration{newAppRegistration(100, "middleware", func(app *fiber.App) {
		registerMiddleware(app, cfg, logger, obs, runtimeMetrics)
	})}
}

func newHealthRoutesRegistration(
	cat catalog.Catalog,
	checks collectionx.List[healthCheckDefinition],
	obs observabilityx.Observability,
) healthRoutesRegistration {
	return healthRoutesRegistration{newAppRegistration(200, "health_routes", func(app *fiber.App) {
		registerHealthRoutes(app, cat, checks, obs)
	})}
}

func newRobotsRouteRegistration(
	cfg *config.Config,
	logger *slog.Logger,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) robotsRouteRegistration {
	return robotsRouteRegistration{newAppRegistration(250, "robots_route", func(app *fiber.App) {
		registerRobotsRoute(app, cfg, logger, cat, bodyCache)
	})}
}

func newAssetRouteRegistration(
	cfg *config.Config,
	runtime assetRouteRuntime,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	bodyCache *assetcache.Cache,
	bus eventx.BusRuntime,
) assetRouteRegistration {
	return assetRouteRegistration{newAppRegistration(300, "asset_route", func(app *fiber.App) {
		registerAssetRoute(app, cfg, runtime.logger, assetResolver, pipelineSvc, bodyCache, bus, runtime.trackDelivery)
	})}
}

func newAssetRouteRuntime(logger *slog.Logger, obs observabilityx.Observability) assetRouteRuntime {
	return assetRouteRuntime{
		logger:        logger,
		trackDelivery: shouldTrackAssetDelivery(logger, obs),
	}
}

func shouldTrackAssetDelivery(logger *slog.Logger, obs observabilityx.Observability) bool {
	return obs != nil || (logger != nil && logger.Enabled(context.Background(), slog.LevelInfo))
}

func newServerRegistrations(
	middleware middlewareRegistration,
	healthRoutes healthRoutesRegistration,
	robotsRoute robotsRouteRegistration,
	assetRoute assetRouteRegistration,
) collectionx.List[appRegistration] {
	return collectionx.NewList(
		middleware.appRegistration,
		healthRoutes.appRegistration,
		robotsRoute.appRegistration,
		assetRoute.appRegistration,
	).Sort(func(left, right appRegistration) int {
		if left.Order != right.Order {
			return cmp.Compare(left.Order, right.Order)
		}
		return cmp.Compare(left.Name, right.Name)
	})
}

func newServerFromDeps(cfg *config.Config, registrations collectionx.List[appRegistration]) (*fiber.App, error) {
	app, err := newServerApp(cfg)
	if err != nil {
		return nil, err
	}
	registrations.Range(func(_ int, registration appRegistration) bool {
		if registration.Apply != nil {
			registration.Apply(app)
		}
		return true
	})
	return app, nil
}
