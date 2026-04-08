package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/oops"
)

func bootstrapCatalogOnStart(
	ctx context.Context,
	runtime catalogBootstrapRuntime,
) error {
	bootstrapErr := oops.In("runtime").Owner("catalog bootstrap")
	startedAt := time.Now()
	totalBytes, scanErr := scanCatalogAssets(runtime.src, runtime.cat)
	if scanErr != nil {
		return scanErr
	}

	warmErr := runtime.pipelineSvc.Warm(ctx)
	if warmErr != nil {
		return bootstrapErr.With("service", "pipeline").Wrap(warmErr)
	}
	cacheStats, cacheErr := runtime.bodyCache.Warm(ctx, runtime.cat)
	if cacheErr != nil {
		return bootstrapErr.With("service", "asset memory cache").Wrap(cacheErr)
	}

	runtime.logger.LogAttrs(
		ctx,
		slog.LevelInfo,
		"Catalog ready",
		catalogReadyAttrs(runtime.cfg, runtime.cat, runtime.bodyCache, cacheStats, totalBytes, time.Since(startedAt)).Values()...,
	)
	return nil
}

func scanCatalogAssets(src source.Source, cat catalog.Catalog) (int64, error) {
	scanErr := oops.In("runtime").Owner("catalog scan")
	totalBytes := int64(0)
	if err := src.Walk(func(file source.File) error {
		if file.IsDir {
			return nil
		}

		asset, err := buildCatalogAsset(file)
		if err != nil {
			return err
		}
		totalBytes += file.Size
		if err := cat.UpsertAsset(asset); err != nil {
			return scanErr.With("asset_path", file.Path).Wrap(err)
		}
		return nil
	}); err != nil {
		return 0, scanErr.Wrap(err)
	}
	return totalBytes, nil
}

func buildCatalogAsset(file source.File) (*catalog.Asset, error) {
	sourceHash, err := pkg.HashFile(file.FullPath)
	if err != nil {
		return nil, oops.In("runtime").Owner("catalog asset").With("asset_path", file.Path).Wrap(err)
	}
	return &catalog.Asset{
		Path:       file.Path,
		FullPath:   file.FullPath,
		Size:       file.Size,
		MediaType:  string(pkg.DetectMIME(file.FullPath)),
		SourceHash: sourceHash,
		ETag:       fmt.Sprintf("%q", sourceHash),
		Metadata: collectionx.NewMapFrom(map[string]string{
			"mtime_unix": strconv.FormatInt(file.ModTime.Unix(), 10),
		}),
	}, nil
}

func logConfigOnStart(ctx context.Context, runtime catalogBootstrapRuntime) error {
	runtime.logger.LogAttrs(ctx, slog.LevelInfo, "Config loaded", configLogAttrs(runtime.cfg).Values()...)
	return nil
}

func catalogReadyAttrs(
	cfg *config.Config,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	cacheStats assetcache.WarmStats,
	totalBytes int64,
	duration time.Duration,
) collectionx.List[slog.Attr] {
	return collectionx.NewList(
		slog.Int("assets", cat.AssetCount()),
		slog.Int("variants", cat.VariantCount()),
		slog.Int64("bytes", totalBytes),
		slog.Bool("memory_cache_enable", bodyCache.Enabled()),
		slog.Bool("memory_cache_warmup", bodyCache.WarmupEnabled()),
		slog.Int("memory_cache_entries", cacheStats.Entries),
		slog.Int64("memory_cache_bytes", cacheStats.Bytes),
		slog.String("compression_mode", cfg.Compression.NormalizedMode()),
		slog.Duration("duration", duration),
	)
}

func configLogAttrs(cfg *config.Config) collectionx.List[slog.Attr] {
	return collectionx.NewList(
		slog.Int("http_port", cfg.HTTP.Port),
		slog.Bool("http_low_memory", cfg.HTTP.LowMemory),
		slog.Bool("http_memory_cache_enable", cfg.HTTP.MemoryCache.Enabled()),
		slog.Bool("http_memory_cache_warmup", cfg.HTTP.MemoryCache.WarmupEnabled()),
		slog.Int("http_memory_cache_max_entries", cfg.HTTP.MemoryCache.MaxEntries),
		slog.Int64("http_memory_cache_max_file_size", cfg.HTTP.MemoryCache.MaxFileSize),
		slog.String("http_memory_cache_ttl", cfg.HTTP.MemoryCache.ParsedTTL().String()),
		slog.String("assets_root", cfg.Assets.Root),
		slog.String("assets_path", cfg.Assets.Path),
		slog.String("assets_backend", string(cfg.Assets.NormalizedBackend())),
		slog.String("assets_entry", cfg.Assets.Entry),
		slog.String("fallback_on", string(cfg.Assets.Fallback.On)),
		slog.String("fallback_target", cfg.Assets.Fallback.Target),
		slog.Int("async_workers", cfg.Async.NormalizedWorkers()),
		slog.Bool("compression_enable", cfg.Compression.Enable),
		slog.String("compression_mode", cfg.Compression.NormalizedMode()),
		slog.String("compression_cache_dir", cfg.Compression.CacheDir),
		slog.Int64("compression_min_size", cfg.Compression.MinSize),
		slog.Int("compression_workers", cfg.Compression.Workers),
		slog.Int("compression_queue_size", cfg.Compression.QueueCapacity()),
		slog.Any("compression_encodings", cfg.Compression.NormalizedEncodings().Values()),
		slog.Int("compression_zstd_level", cfg.Compression.ZstdLevel),
		slog.Bool("image_enable", cfg.Image.Enable),
		slog.Any("image_widths", cfg.Image.ParsedWidths().Values()),
		slog.Int("image_jpeg_quality", cfg.Image.JPEGQuality),
		slog.Bool("debug_enable", cfg.Debug.Enable),
		slog.Int("debug_live_port", cfg.Debug.LivePort),
		slog.String("metrics_prefix", cfg.Metrics.Prefix),
		slog.String("logger_level", cfg.Logger.Level),
	)
}
