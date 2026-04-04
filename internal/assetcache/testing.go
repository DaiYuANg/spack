package assetcache

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/config"
)

// NewCacheForTest exposes cache construction for external tests.
func NewCacheForTest(cfg config.MemoryCache, logger *slog.Logger) *Cache {
	return NewCacheWithObservabilityForTest(cfg, logger, nil)
}

// NewCacheWithObservabilityForTest exposes cache construction with observability for external tests.
func NewCacheWithObservabilityForTest(
	cfg config.MemoryCache,
	logger *slog.Logger,
	obs observabilityx.Observability,
) *Cache {
	return newCache(&config.HTTP{MemoryCache: cfg}, logger, obs, nil)
}

// NewCacheWithBusForTest exposes cache construction with an event bus for external tests.
func NewCacheWithBusForTest(
	cfg config.MemoryCache,
	logger *slog.Logger,
	obs observabilityx.Observability,
	bus eventx.BusRuntime,
) *Cache {
	return newCache(&config.HTTP{MemoryCache: cfg}, logger, obs, bus)
}

// StartForTest exposes cache lifecycle start for external tests.
func StartForTest(cache *Cache) error {
	return cache.start(context.Background())
}
