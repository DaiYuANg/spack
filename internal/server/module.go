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
	dix.WithModuleProviders(
		dix.Provider4(newServerRuntimeDeps),
		dix.Provider4(newServerFromDeps),
	),
)

type serverRuntimeDeps struct {
	bodyCache     *assetcache.Cache
	assetResolver *resolver.Resolver
	pipelineSvc   *pipeline.Service
	obs           observabilityx.Observability
}

func newServerRuntimeDeps(
	bodyCache *assetcache.Cache,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	obs observabilityx.Observability,
) serverRuntimeDeps {
	return serverRuntimeDeps{
		bodyCache:     bodyCache,
		assetResolver: assetResolver,
		pipelineSvc:   pipelineSvc,
		obs:           obs,
	}
}

func newServerFromDeps(
	cfg *config.Config,
	logger *slog.Logger,
	cat catalog.Catalog,
	deps serverRuntimeDeps,
) *fiber.App {
	return newServer(cfg, logger, cat, deps.bodyCache, deps.assetResolver, deps.pipelineSvc, deps.obs)
}
