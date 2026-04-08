package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/mo"
)

type debugRuntime struct {
	enabled    bool
	server     *http.Server
	liveURL    string
	metricsURL string
	statsviz   string
}

func startHTTPRuntime(_ context.Context, runtime httpRuntime) error {
	go func() {
		address := "127.0.0.1:" + runtime.cfg.HTTP.GetPort()
		runtime.logger.Info("HTTP runtime listening",
			slog.String("address", "http://"+address),
			slog.String("mount_path", runtime.cfg.Assets.Path),
			slog.Int("assets", runtime.cat.AssetCount()),
			slog.Int("variants", runtime.cat.VariantCount()),
		)
		if err := runtime.app.Listen(":"+runtime.cfg.HTTP.GetPort(), fiber.ListenConfig{
			DisableStartupMessage: true,
		}); err != nil {
			runtime.logger.Error("HTTP runtime stopped", slog.String("err", err.Error()))
		}
	}()
	return nil
}

func stopHTTPRuntime(ctx context.Context, runtime httpRuntime) error {
	return runtime.app.ShutdownWithContext(ctx)
}

func buildDebugRuntime(
	cfg *config.Config,
	logger *slog.Logger,
	pipelineMetrics *pipeline.Metrics,
	metricsAdapter *obsprom.Adapter,
) *debugRuntime {
	if !cfg.Debug.Enable {
		return &debugRuntime{}
	}

	mux := http.NewServeMux()
	if pipelineMetrics != nil {
		prometheus.MustRegister(pipelineMetrics.Collectors()...)
	}
	mux.Handle(cfg.Metrics.Prefix, metricsAdapter.Handler())
	if err := statsviz.Register(mux); err != nil {
		logger.Error("Debug runtime registration failed", slog.String("err", err.Error()))
		return &debugRuntime{}
	}

	address := fmt.Sprintf(cfg.Debug.Address+":%d", cfg.Debug.LivePort)
	return &debugRuntime{
		enabled: true,
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		liveURL:    fmt.Sprintf("http://127.0.0.1:%d", cfg.Debug.LivePort),
		metricsURL: fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Debug.LivePort, cfg.Metrics.Prefix),
		statsviz:   fmt.Sprintf("http://127.0.0.1:%d/debug/statsviz", cfg.Debug.LivePort),
	}
}

func startDebugRuntime(_ context.Context, logger *slog.Logger, runtime *debugRuntime) error {
	server, ok := debugServer(runtime).Get()
	if !ok {
		return nil
	}

	go func() {
		logger.Info("Debug runtime listening",
			slog.String("address", runtime.liveURL),
			slog.String("metrics", runtime.metricsURL),
			slog.String("statsviz", runtime.statsviz),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Debug runtime stopped", slog.String("err", err.Error()))
		}
	}()
	return nil
}

func stopDebugRuntime(ctx context.Context, runtime *debugRuntime) error {
	server, ok := debugServer(runtime).Get()
	if !ok {
		return nil
	}
	return server.Shutdown(ctx)
}

func startRuntime(ctx context.Context, bootstrap catalogBootstrapRuntime, http httpRuntime, debug *debugRuntime) error {
	if err := logConfigOnStart(ctx, bootstrap); err != nil {
		return err
	}
	if err := bootstrapCatalogOnStart(ctx, bootstrap); err != nil {
		return err
	}
	if err := startHTTPRuntime(ctx, http); err != nil {
		return err
	}
	return startDebugRuntime(ctx, bootstrap.logger, debug)
}

func stopRuntime(ctx context.Context, http httpRuntime, debug *debugRuntime) error {
	if err := stopDebugRuntime(ctx, debug); err != nil {
		return err
	}
	return stopHTTPRuntime(ctx, http)
}

func debugServer(runtime *debugRuntime) mo.Option[*http.Server] {
	if runtime == nil || !runtime.enabled || runtime.server == nil {
		return mo.None[*http.Server]()
	}
	return mo.Some(runtime.server)
}
