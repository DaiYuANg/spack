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

	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

func bootstrapCatalog(
	lc fx.Lifecycle,
	cfg *config.Config,
	src source.Source,
	cat catalog.Catalog,
	pipelineSvc *pipeline.Service,
	logger *slog.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
		},
	})
}

func httpLifecycle(
	lc fx.Lifecycle,
	app *fiber.App,
	cfg *config.Config,
	cat catalog.Catalog,
	logger *slog.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	})
}

func debugLifecycle(
	lc fx.Lifecycle,
	cfg *config.Config,
	logger *slog.Logger,
	httpRequestsTotal *prometheus.CounterVec,
	httpRequestDurationSeconds *prometheus.HistogramVec,
	activeRequests *prometheus.GaugeVec,
	pipelineMetrics *pipeline.Metrics,
) {
	if !cfg.Debug.Enable {
		return
	}

	mux := http.NewServeMux()
	collectors := []prometheus.Collector{
		httpRequestsTotal,
		httpRequestDurationSeconds,
		activeRequests,
	}
	collectors = append(collectors, pipelineMetrics.Collectors()...)
	prometheus.MustRegister(collectors...)
	mux.Handle(cfg.Metrics.Prefix, promhttp.Handler())
	if err := statsviz.Register(mux); err != nil {
		logger.Error("Debug runtime registration failed", slog.String("err", err.Error()))
		return
	}

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Debug.LivePort),
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
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
