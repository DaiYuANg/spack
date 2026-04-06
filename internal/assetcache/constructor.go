package assetcache

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/config"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/hot"
)

type Cache struct {
	logger *slog.Logger
	obs    observabilityx.Observability
	policy cachepolicy.MemoryPolicy
	warmup bool
	cache  *hot.HotCache[string, []byte]
	bus    eventx.BusRuntime
	pool   *ants.Pool

	variantRemovedUnsubscribe   func()
	variantGeneratedUnsubscribe func()
}

func newCache(cfg *config.HTTP, logger *slog.Logger, obs observabilityx.Observability, bus eventx.BusRuntime, pool *ants.Pool) *Cache {
	cacheCfg := cfg.MemoryCache
	cache := &Cache{
		logger: logger,
		obs:    observabilityx.Normalize(obs, logger),
		policy: cachepolicy.NewMemoryPolicy(cacheCfg),
		warmup: cacheCfg.WarmupEnabled(),
		bus:    bus,
		pool:   pool,
	}
	if !cacheCfg.Enabled() {
		return cache
	}

	cache.cache = hot.NewHotCache[string, []byte](hot.WTinyLFU, cacheCfg.MaxEntries).
		WithTTL(cacheCfg.ParsedTTL()).
		WithJanitor().
		WithEvictionCallback(cache.onEviction).
		Build()
	return cache
}
