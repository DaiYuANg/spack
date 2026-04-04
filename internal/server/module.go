package server

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

var Module = dix.NewModule("server",
	dix.WithModuleSetups(
		dix.SetupWithMetadata(setupServer, dix.SetupMetadata{
			Label: "SetupServer",
			Dependencies: dix.ServiceRefs(
				dix.TypedService[*config.Config](),
				dix.TypedService[*slog.Logger](),
				dix.TypedService[catalog.Catalog](),
				dix.TypedService[*assetcache.Cache](),
				dix.TypedService[*resolver.Resolver](),
				dix.TypedService[*pipeline.Service](),
				dix.TypedService[observabilityx.Observability](),
			),
			Provides: dix.ServiceRefs(
				dix.TypedService[*fiber.App](),
			),
			GraphMutation: true,
		}),
	),
)

func setupServer(c *dix.Container, _ dix.Lifecycle) error {
	cfg, err := dix.ResolveAs[*config.Config](c)
	if err != nil {
		return err
	}
	logger, err := dix.ResolveAs[*slog.Logger](c)
	if err != nil {
		return err
	}
	cat, err := dix.ResolveAs[catalog.Catalog](c)
	if err != nil {
		return err
	}
	bodyCache, err := dix.ResolveAs[*assetcache.Cache](c)
	if err != nil {
		return err
	}
	assetResolver, err := dix.ResolveAs[*resolver.Resolver](c)
	if err != nil {
		return err
	}
	pipelineSvc, err := dix.ResolveAs[*pipeline.Service](c)
	if err != nil {
		return err
	}
	obs, err := dix.ResolveAs[observabilityx.Observability](c)
	if err != nil {
		return err
	}

	dix.ProvideValueT(c, newServer(cfg, logger, cat, bodyCache, assetResolver, pipelineSvc, obs))
	return nil
}
