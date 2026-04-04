package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

func bootstrapCatalog(
	lc dix.Lifecycle,
	cfg *config.Config,
	src source.Source,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	pipelineSvc *pipeline.Service,
	logger *slog.Logger,
) {
	lc.OnStart(func(ctx context.Context) error {
		startedAt := time.Now()
		totalBytes, scanErr := scanCatalogAssets(src, cat)
		if scanErr != nil {
			return scanErr
		}

		warmErr := pipelineSvc.Warm(ctx)
		if warmErr != nil {
			return fmt.Errorf("warm pipeline: %w", warmErr)
		}
		cacheStats, cacheErr := bodyCache.Warm(ctx, cat)
		if cacheErr != nil {
			return fmt.Errorf("warm asset memory cache: %w", cacheErr)
		}

		logger.LogAttrs(
			ctx,
			slog.LevelInfo,
			"Catalog ready",
			catalogReadyAttrs(cfg, cat, bodyCache, cacheStats, totalBytes, time.Since(startedAt)).Values()...,
		)
		return nil
	})
}

func scanCatalogAssets(src source.Source, cat catalog.Catalog) (int64, error) {
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
			return fmt.Errorf("upsert asset %s: %w", file.Path, err)
		}
		return nil
	}); err != nil {
		return 0, fmt.Errorf("walk source assets: %w", err)
	}
	return totalBytes, nil
}

func buildCatalogAsset(file source.File) (*catalog.Asset, error) {
	sourceHash, err := hashFile(file.FullPath)
	if err != nil {
		return nil, fmt.Errorf("hash asset %s: %w", file.Path, err)
	}
	return &catalog.Asset{
		Path:       file.Path,
		FullPath:   file.FullPath,
		Size:       file.Size,
		MediaType:  string(pkg.DetectMIME(file.FullPath)),
		SourceHash: sourceHash,
		ETag:       fmt.Sprintf("%q", sourceHash),
		Metadata: map[string]string{
			"mtime_unix": strconv.FormatInt(file.ModTime.Unix(), 10),
		},
	}, nil
}

func logConfigLifecycle(lc dix.Lifecycle, cfg *config.Config, logger *slog.Logger) {
	lc.OnStart(func(ctx context.Context) error {
		logger.LogAttrs(ctx, slog.LevelInfo, "Config loaded", configLogAttrs(cfg).Values()...)
		return nil
	})
}

func httpLifecycle(
	lc dix.Lifecycle,
	app *fiber.App,
	cfg *config.Config,
	cat catalog.Catalog,
	logger *slog.Logger,
) {
	lc.OnStart(func(ctx context.Context) error {
		go func() {
			address := "127.0.0.1:" + cfg.HTTP.GetPort()
			logger.Info("HTTP runtime listening",
				slog.String("address", "http://"+address),
				slog.String("mount_path", cfg.Assets.Path),
				slog.Int("assets", cat.AssetCount()),
				slog.Int("variants", cat.VariantCount()),
			)
			if err := app.Listen(":"+cfg.HTTP.GetPort(), fiber.ListenConfig{
				DisableStartupMessage: true,
			}); err != nil {
				logger.Error("HTTP runtime stopped", slog.String("err", err.Error()))
			}
		}()
		return nil
	})

	lc.OnStop(func(ctx context.Context) error {
		return app.ShutdownWithContext(ctx)
	})
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
		slog.Bool("compression_enable", cfg.Compression.Enable),
		slog.String("compression_mode", cfg.Compression.NormalizedMode()),
		slog.String("compression_cache_dir", cfg.Compression.CacheDir),
		slog.Int64("compression_min_size", cfg.Compression.MinSize),
		slog.Int("compression_workers", cfg.Compression.Workers),
		slog.Int("compression_queue_size", cfg.Compression.QueueCapacity()),
		slog.Bool("image_enable", cfg.Image.Enable),
		slog.Any("image_widths", cfg.Image.ParsedWidths().Values()),
		slog.Int("image_jpeg_quality", cfg.Image.JPEGQuality),
		slog.Bool("debug_enable", cfg.Debug.Enable),
		slog.Int("debug_live_port", cfg.Debug.LivePort),
		slog.String("metrics_prefix", cfg.Metrics.Prefix),
		slog.String("logger_level", cfg.Logger.Level),
	)
}

func debugLifecycle(
	lc dix.Lifecycle,
	cfg *config.Config,
	logger *slog.Logger,
	pipelineMetrics *pipeline.Metrics,
	metricsAdapter *obsprom.Adapter,
) {
	if !cfg.Debug.Enable {
		return
	}

	mux := http.NewServeMux()
	if pipelineMetrics != nil {
		prometheus.MustRegister(pipelineMetrics.Collectors()...)
	}
	mux.Handle(cfg.Metrics.Prefix, metricsAdapter.Handler())
	if err := statsviz.Register(mux); err != nil {
		logger.Error("Debug runtime registration failed", slog.String("err", err.Error()))
		return
	}

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", cfg.Debug.LivePort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.OnStart(func(ctx context.Context) error {
		go func() {
			logger.Info("Debug runtime listening",
				slog.String("address", fmt.Sprintf("http://127.0.0.1:%d", cfg.Debug.LivePort)),
				slog.String("metrics", fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Debug.LivePort, cfg.Metrics.Prefix)),
				slog.String("statsviz", fmt.Sprintf("http://127.0.0.1:%d/debug/statsviz", cfg.Debug.LivePort)),
			)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Debug runtime stopped", slog.String("err", err.Error()))
			}
		}()
		return nil
	})

	lc.OnStop(func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})
}

func hashFile(path string) (string, error) {
	// #nosec G304 -- paths come from the scanned local asset tree.
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hashing: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			return
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("copy file into hasher: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
