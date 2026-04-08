package task

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/oops"
)

const staticRobotsAssetPath = "robots.txt"

type CacheWarmerReport struct {
	Assets        int
	Variants      int
	LoadedEntries int
	LoadedBytes   int64
}

func registerCacheWarmerTask(ctx context.Context, scheduler gocron.Scheduler, runtime *cacheWarmerRuntime) (bool, error) {
	if runtime == nil || runtime.bodyCache == nil || !runtime.bodyCache.Enabled() {
		return false, nil
	}

	interval := cacheWarmerInterval(runtime.cfg)
	job, err := scheduler.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(func() {
			runCacheWarmer(ctx, runtime)
		}),
	)
	if err != nil {
		return false, oops.In("task").Owner("cache warmer").Wrap(err)
	}

	runtime.logger.Info("Task cache warmer enabled",
		slog.String("id", job.ID().String()),
		slog.String("interval", interval.String()),
	)
	return true, nil
}

func runCacheWarmer(ctx context.Context, runtime *cacheWarmerRuntime) {
	startedAt := time.Now()
	report, err := warmCacheHotset(ctx, runtime.cfg, runtime.catalog, runtime.bodyCache)
	recordTaskRunMetrics(ctx, runtime.obs, "cache_warmer", startedAt, err)
	if err != nil {
		runtime.logger.Error("Task cache warmer failed", slog.String("err", err.Error()))
		return
	}
	recordCacheWarmerMetrics(ctx, runtime.obs, report)
	if report.LoadedEntries == 0 {
		return
	}

	runtime.logger.Info("Task cache warmer completed",
		slog.Int("assets", report.Assets),
		slog.Int("variants", report.Variants),
		slog.Int("loaded_entries", report.LoadedEntries),
		slog.Int64("loaded_bytes", report.LoadedBytes),
		slog.Duration("duration", time.Since(startedAt)),
	)
}

func warmCacheHotset(
	ctx context.Context,
	cfg *config.Config,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (CacheWarmerReport, error) {
	if cfg == nil || cat == nil || bodyCache == nil || !bodyCache.Enabled() {
		return CacheWarmerReport{}, nil
	}

	assets := collectWarmAssets(cfg, cat)
	report := buildCacheWarmerReport(cat, assets)
	if report.Assets == 0 {
		return report, nil
	}

	hotset, err := buildHotsetCatalog(ctx, cat, assets)
	if err != nil {
		return CacheWarmerReport{}, err
	}
	stats, err := bodyCache.WarmSelected(ctx, hotset)
	if err != nil {
		return CacheWarmerReport{}, oops.In("task").Owner("cache warmer").Wrap(err)
	}

	report.LoadedEntries = stats.Entries
	report.LoadedBytes = stats.Bytes
	return report, nil
}

func collectWarmAssets(cfg *config.Config, cat catalog.Catalog) collectionx.List[*catalog.Asset] {
	assetPaths := collectionx.NewOrderedSet[string]()
	assetPaths.Add(strings.TrimSpace(cfg.Assets.Entry))
	assetPaths.Add(strings.TrimSpace(cfg.Assets.Fallback.Target))
	if cfg.Robots.Enable {
		assetPaths.Add(staticRobotsAssetPath)
	}

	assets := collectionx.NewList[*catalog.Asset]()
	for _, assetPath := range assetPaths.Values() {
		if strings.TrimSpace(assetPath) == "" {
			continue
		}
		if asset, ok := cat.FindAsset(assetPath); ok && asset != nil {
			assets.Add(asset)
		}
	}
	return assets
}

func buildHotsetCatalog(
	ctx context.Context,
	cat catalog.Catalog,
	assets collectionx.List[*catalog.Asset],
) (catalog.Catalog, error) {
	hotset := catalog.NewInMemoryCatalog()
	for _, asset := range assets.Values() {
		if err := copyHotsetAsset(ctx, hotset, cat, asset); err != nil {
			return nil, err
		}
	}
	return hotset, nil
}

func copyHotsetAsset(
	ctx context.Context,
	hotset catalog.Catalog,
	cat catalog.Catalog,
	asset *catalog.Asset,
) error {
	if err := cacheWarmerContextErr(ctx); err != nil {
		return err
	}
	if asset == nil {
		return nil
	}
	if err := hotset.UpsertAsset(asset); err != nil {
		return oops.In("task").Owner("cache warmer").With("asset_path", asset.Path).Wrap(err)
	}
	return copyHotsetVariants(ctx, hotset, cat.ListVariants(asset.Path))
}

func copyHotsetVariants(
	ctx context.Context,
	hotset catalog.Catalog,
	variants collectionx.List[*catalog.Variant],
) error {
	for _, variant := range variants.Values() {
		if err := cacheWarmerContextErr(ctx); err != nil {
			return err
		}
		if variant == nil {
			continue
		}
		if err := hotset.UpsertVariant(variant); err != nil {
			return oops.In("task").Owner("cache warmer").With("artifact_path", variant.ArtifactPath).Wrap(err)
		}
	}
	return nil
}

func buildCacheWarmerReport(cat catalog.Catalog, assets collectionx.List[*catalog.Asset]) CacheWarmerReport {
	report := CacheWarmerReport{Assets: assets.Len()}
	assets.Range(func(_ int, asset *catalog.Asset) bool {
		if asset != nil {
			report.Variants += cat.ListVariants(asset.Path).Len()
		}
		return true
	})
	return report
}

func cacheWarmerInterval(cfg *config.Config) time.Duration {
	if cfg == nil {
		return 2 * time.Minute
	}

	ttl := cfg.HTTP.MemoryCache.ParsedTTL() / 2
	if ttl < time.Minute {
		return time.Minute
	}
	if ttl > 15*time.Minute {
		return 15 * time.Minute
	}
	return ttl
}

func cacheWarmerContextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return oops.In("task").Owner("cache warmer").Wrap(err)
	}
	return nil
}
