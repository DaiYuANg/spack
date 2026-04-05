// Package assetcache caches hot static asset bodies in memory for small-file delivery.
package assetcache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/workerpool"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/hot"
	"github.com/samber/hot/pkg/base"
)

const (
	metricAssetCacheHits         = "asset_cache_hits_total"
	metricAssetCacheMisses       = "asset_cache_misses_total"
	metricAssetCacheFills        = "asset_cache_fills_total"
	metricAssetCacheFillBytes    = "asset_cache_fill_bytes_total"
	metricAssetCacheWarmEntries  = "asset_cache_warm_entries_total"
	metricAssetCacheWarmBytes    = "asset_cache_warm_bytes_total"
	metricAssetCacheEvictions    = "asset_cache_evictions_total"
	metricAssetCacheEvictedBytes = "asset_cache_evicted_bytes_total"
	metricAssetCacheLoadErrors   = "asset_cache_load_errors_total"
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

type WarmStats struct {
	Entries int
	Bytes   int64
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

func (c *Cache) Enabled() bool {
	return c != nil && c.cache != nil
}

func (c *Cache) WarmupEnabled() bool {
	return c.Enabled() && c.warmup
}

func (c *Cache) ShouldServe(size int64, rangeRequested bool) bool {
	if !c.Enabled() || c.policy == nil {
		return false
	}
	return c.policy.ShouldServe(size, rangeRequested)
}

func (c *Cache) GetOrLoad(path string) ([]byte, bool, error) {
	if !c.Enabled() {
		return nil, false, errors.New("memory cache is disabled")
	}

	if body, found := c.cache.MustGet(path); found {
		c.addCounter(metricAssetCacheHits, 1)
		return body, true, nil
	}
	c.addCounter(metricAssetCacheMisses, 1)

	body, err := c.readAndCachePath(path)
	if err != nil {
		c.addCounter(metricAssetCacheLoadErrors, 1)
		return nil, false, err
	}

	c.addCounter(metricAssetCacheFills, 1)
	c.addCounter(metricAssetCacheFillBytes, int64(len(body)))
	return body, false, nil
}

func (c *Cache) Delete(path string) bool {
	if !c.Enabled() {
		return false
	}
	return c.cache.Delete(path)
}

func (c *Cache) Warm(ctx context.Context, cat catalog.Catalog) (WarmStats, error) {
	if !c.WarmupEnabled() {
		return WarmStats{}, nil
	}

	stats := WarmStats{}
	if err := c.warmAssets(ctx, cat, &stats); err != nil {
		return WarmStats{}, fmt.Errorf("warm memory cache: %w", err)
	}
	c.recordWarmStats(ctx, stats)

	return stats, nil
}

func (c *Cache) warmAssets(ctx context.Context, cat catalog.Catalog, stats *WarmStats) error {
	var statsMu sync.Mutex
	err := workerpool.RunList(ctx, c.pool, cat.AllAssets(), func(ctx context.Context, asset *catalog.Asset) error {
		assetStats := WarmStats{}
		if err := c.warmAsset(ctx, cat, asset, &assetStats); err != nil {
			return err
		}
		if assetStats.Entries == 0 && assetStats.Bytes == 0 {
			return nil
		}
		statsMu.Lock()
		stats.Entries += assetStats.Entries
		stats.Bytes += assetStats.Bytes
		statsMu.Unlock()
		return nil
	})
	if err != nil {
		return fmt.Errorf("run asset warm list: %w", err)
	}
	return nil
}

func (c *Cache) warmAsset(
	ctx context.Context,
	cat catalog.Catalog,
	asset *catalog.Asset,
	stats *WarmStats,
) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := c.preloadPath(asset.FullPath, asset.Size, stats); err != nil {
		return err
	}
	return c.warmVariants(ctx, cat.ListVariants(asset.Path), stats)
}

func (c *Cache) warmVariants(
	ctx context.Context,
	variants collectionx.List[*catalog.Variant],
	stats *WarmStats,
) error {
	return warmListSerial[*catalog.Variant](ctx, variants, stats, func(ctx context.Context, variant *catalog.Variant, stats *WarmStats) error {
		return c.warmVariant(ctx, variant, stats)
	})
}

func warmListSerial[T any](
	ctx context.Context,
	values collectionx.List[T],
	stats *WarmStats,
	warm func(context.Context, T, *WarmStats) error,
) error {
	var warmErr error
	values.Range(func(_ int, value T) bool {
		warmErr = warm(ctx, value, stats)
		return warmErr == nil
	})
	return pickWarmError(ctx, warmErr)
}

func (c *Cache) warmVariant(ctx context.Context, variant *catalog.Variant, stats *WarmStats) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	return c.preloadPath(variant.ArtifactPath, variant.Size, stats)
}

func pickWarmError(ctx context.Context, warmErr error) error {
	if warmErr != nil {
		return warmErr
	}
	return contextErr(ctx)
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context done: %w", ctx.Err())
	default:
		return nil
	}
}

func (c *Cache) preloadPath(path string, size int64, stats *WarmStats) error {
	if !c.ShouldServe(size, false) || strings.TrimSpace(path) == "" {
		return nil
	}
	if _, found := c.cache.MustGet(path); found {
		return nil
	}

	body, err := c.readAndCachePath(path)
	if err != nil {
		return err
	}

	if stats != nil {
		stats.Entries++
		stats.Bytes += int64(len(body))
	}
	return nil
}

func (c *Cache) readAndCachePath(path string) ([]byte, error) {
	body, err := c.readFile(path)
	if err != nil {
		return nil, err
	}

	c.cache.Set(path, body)
	return body, nil
}

func (c *Cache) readFile(path string) ([]byte, error) {
	// #nosec G304 -- path comes from resolver/catalog-selected asset paths already validated against the asset tree.
	body, err := os.ReadFile(path)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Read asset failed",
				slog.String("path", path),
				slog.String("err", err.Error()),
			)
		}
		return nil, fmt.Errorf("read resolved asset: %w", err)
	}
	return body, nil
}

func (c *Cache) recordWarmStats(ctx context.Context, stats WarmStats) {
	if stats.Entries == 0 {
		return
	}
	c.addCounterWithContext(ctx, metricAssetCacheWarmEntries, int64(stats.Entries))
	c.addCounterWithContext(ctx, metricAssetCacheWarmBytes, stats.Bytes)
}

func (c *Cache) onEviction(_ base.EvictionReason, _ string, body []byte) {
	c.addCounter(metricAssetCacheEvictions, 1)
	c.addCounter(metricAssetCacheEvictedBytes, int64(len(body)))
}

func (c *Cache) addCounter(name string, value int64) {
	c.addCounterWithContext(context.Background(), name, value)
}

func (c *Cache) addCounterWithContext(ctx context.Context, name string, value int64) {
	if value == 0 || c == nil || c.obs == nil {
		return
	}
	c.obs.AddCounter(ctx, name, value)
}
