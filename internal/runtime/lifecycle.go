package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/dix"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

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

	address := fmt.Sprintf(cfg.Debug.Address+":%d", cfg.Debug.LivePort)
	logger.Info("Debug runtime listening", slog.String("address", address))
	server := &http.Server{
		Addr:              address,
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
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("Debug runtime stopped", slog.String("err", err.Error()))
			}
		}()
		return nil
	})

	lc.OnStop(func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})
}
