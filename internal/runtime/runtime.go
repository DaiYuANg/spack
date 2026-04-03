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
	"time"

	"github.com/DaiYuANg/arcgo/dix"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/arl/statsviz"
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
	pipelineSvc *pipeline.Service,
	logger *slog.Logger,
) {
	lc.OnStart(func(ctx context.Context) error {
		startedAt := time.Now()
		totalBytes := int64(0)

		if err := src.Walk(func(file source.File) error {
			if file.IsDir {
				return nil
			}

			sourceHash, err := hashFile(file.FullPath)
			if err != nil {
				return err
			}

			asset := &catalog.Asset{
				Path:       file.Path,
				FullPath:   file.FullPath,
				Size:       file.Size,
				MediaType:  string(pkg.DetectMIME(file.FullPath)),
				SourceHash: sourceHash,
				ETag:       fmt.Sprintf("\"%s\"", sourceHash),
				Metadata: map[string]string{
					"mtime_unix": fmt.Sprintf("%d", file.ModTime.Unix()),
				},
			}
			totalBytes += file.Size
			return cat.UpsertAsset(asset)
		}); err != nil {
			return err
		}

		if err := pipelineSvc.Warm(ctx); err != nil {
			return err
		}

		logger.Info("Catalog ready",
			slog.Int("assets", cat.AssetCount()),
			slog.Int("variants", cat.VariantCount()),
			slog.Int64("bytes", totalBytes),
			slog.String("compression_mode", cfg.Compression.NormalizedMode()),
			slog.Duration("duration", time.Since(startedAt)),
		)
		return nil
	})
}

func logConfigLifecycle(lc dix.Lifecycle, cfg *config.Config, logger *slog.Logger) {
	lc.OnStart(func(_ context.Context) error {
		logger.Info("Config loaded",
			slog.Int("http_port", cfg.Http.Port),
			slog.Bool("http_low_memory", cfg.Http.LowMemory),
			slog.String("assets_root", cfg.Assets.Root),
			slog.String("assets_path", cfg.Assets.Path),
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
			address := "127.0.0.1:" + cfg.Http.GetPort()
			logger.Info("HTTP runtime listening",
				slog.String("address", "http://"+address),
				slog.String("mount_path", cfg.Assets.Path),
				slog.Int("assets", cat.AssetCount()),
				slog.Int("variants", cat.VariantCount()),
			)
			if err := app.Listen(":" + cfg.Http.GetPort()); err != nil {
				logger.Error("HTTP runtime stopped", slog.String("err", err.Error()))
			}
		}()
		return nil
	})

	lc.OnStop(func(ctx context.Context) error {
		return app.ShutdownWithContext(ctx)
	})
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
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Debug.LivePort),
		Handler: mux,
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
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
