package assetcache

import (
	"log/slog"

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
	return newCache(&config.HTTP{MemoryCache: cfg}, logger, obs)
}
