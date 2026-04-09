package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/media"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/daiyuang/spack/view"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/etag"
	expvarmw "github.com/gofiber/fiber/v3/middleware/expvar"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/template/html/v2"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

var (
	httpRequestsTotalSpec = observabilityx.NewCounterSpec(
		"http_requests_total",
		observabilityx.WithDescription("Total number of HTTP requests handled by the Fiber server."),
		observabilityx.WithLabelKeys("method", "path", "status"),
	)
	httpRequestDurationSpec = observabilityx.NewHistogramSpec(
		"http_request_duration_seconds",
		observabilityx.WithDescription("HTTP request duration in seconds."),
		observabilityx.WithUnit("s"),
		observabilityx.WithLabelKeys("method", "path", "status"),
	)
	httpAssetDeliveryTotalSpec = observabilityx.NewCounterSpec(
		"http_asset_delivery_total",
		observabilityx.WithDescription("Total number of asset delivery responses by delivery path."),
		observabilityx.WithLabelKeys("method", "path", "status", "delivery"),
	)
	httpAssetDeliveryDurationSpec = observabilityx.NewHistogramSpec(
		"http_asset_delivery_duration_seconds",
		observabilityx.WithDescription("Asset delivery request duration in seconds."),
		observabilityx.WithUnit("s"),
		observabilityx.WithLabelKeys("method", "path", "status", "delivery"),
	)
)

func newServerApp(cfg *config.Config) (*fiber.App, error) {
	header, err := buildServerHeader()
	if err != nil {
		return nil, oops.In("server").Owner("app").Wrap(err)
	}
	return fiber.New(fiber.Config{
		AppName:           "Spack",
		Views:             html.NewFileSystem(http.FS(view.View), ".html"),
		PassLocalsToViews: true,
		Immutable:         true,
		StreamRequestBody: true,
		ErrorHandler:      errorHandler,
		ServerHeader:      header,
		StrictRouting:     true,
		ReduceMemoryUsage: cfg.HTTP.LowMemory,
	}), nil
}

func registerMiddleware(
	app *fiber.App,
	cfg *config.Config,
	logger *slog.Logger,
	obs observabilityx.Observability,
	runtimeMetrics *RuntimeMetrics,
) {
	requestIDConfig := requestid.ConfigDefault
	requestIDConfig.Header = RequestIDHeader
	requestIDConfig.Generator = requestIDGenerator
	app.Use(requestid.New(requestIDConfig))
	app.Use(etag.New())
	app.Use(helmet.New())
	app.Use(requestLogMiddleware(logger))
	app.Use(metricsMiddleware(obs, runtimeMetrics))

	if cfg.Debug.Enable {
		app.Use(expvarmw.New())
		app.Use(pprof.New(pprof.Config{Prefix: cfg.Debug.PprofPrefix}))
	}

	recoverConfig := recoverer.ConfigDefault
	recoverConfig.EnableStackTrace = true
	app.Use(recoverer.New(recoverConfig))
}

func registerAssetRoute(
	app *fiber.App,
	cfg *config.Config,
	logger *slog.Logger,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	bodyCache *assetcache.Cache,
	bus eventx.BusRuntime,
) {
	app.Use(routePattern(cfg.Assets.Path), func(c fiber.Ctx) error {
		requestedFormat := media.NormalizeImageFormat(c.Query("format"))
		request := buildResolverRequest(c, cfg.Assets.Path, requestedFormat)
		result, err := assetResolver.Resolve(request)
		if err != nil {
			return fiber.ErrNotFound
		}

		enqueuePipelineResult(result, pipelineSvc)
		delivery, err := sendResolvedAsset(c, cfg, request, result, requestedFormat, logger, bodyCache)
		if err != nil {
			return err
		}
		if delivery != "" {
			setAssetDelivery(c, delivery)
			publishVariantServed(c.Context(), result, bus, logger)
		}
		return nil
	})
}

func buildResolverRequest(c fiber.Ctx, mountPath, requestedFormat string) resolver.Request {
	return resolver.Request{
		Path:           trimMountPath(c.Path(), mountPath),
		Accept:         c.Get(fiber.HeaderAccept),
		AcceptEncoding: c.Get(fiber.HeaderAcceptEncoding),
		Width:          parsePositiveInt(c.Query("w")),
		Format:         requestedFormat,
		RangeRequested: strings.TrimSpace(c.Get(fiber.HeaderRange)) != "",
	}
}

func enqueuePipelineResult(result *resolver.Result, pipelineSvc *pipeline.Service) {
	if pipelineSvc == nil || result == nil || result.Asset == nil {
		return
	}
	if result.PreferredEncodings.Len() == 0 && result.PreferredWidths.Len() == 0 && result.PreferredFormats.Len() == 0 {
		return
	}

	pipelineSvc.Enqueue(pipeline.Request{
		AssetPath:          result.Asset.Path,
		PreferredEncodings: result.PreferredEncodings,
		PreferredWidths:    result.PreferredWidths,
		PreferredFormats:   result.PreferredFormats,
	})
}

func metricsMiddleware(obs observabilityx.Observability, runtimeMetrics *RuntimeMetrics) fiber.Handler {
	obs = lo.Ternary(obs != nil, obs, observabilityx.NopWithLogger(nil))

	return func(c fiber.Ctx) error {
		runtimeMetrics.IncRequestsInFlight()
		defer runtimeMetrics.DecRequestsInFlight()

		startedAt := time.Now()
		err := c.Next()
		duration := time.Since(startedAt).Seconds()

		requestAttrs := requestMetricsAttrs(c)
		obs.Counter(httpRequestsTotalSpec).Add(context.Background(), 1, requestAttrs...)
		obs.Histogram(httpRequestDurationSpec).Record(context.Background(), duration, requestAttrs...)

		deliveryAttrs := assetDeliveryMetricsAttrs(c)
		if len(deliveryAttrs) > 0 {
			obs.Counter(httpAssetDeliveryTotalSpec).Add(context.Background(), 1, deliveryAttrs...)
			obs.Histogram(httpAssetDeliveryDurationSpec).Record(context.Background(), duration, deliveryAttrs...)
		}
		if err != nil {
			return oops.In("server").Wrap(fmt.Errorf("run metrics middleware chain: %w", err))
		}
		return nil
	}
}

func requestMetricsAttrs(c fiber.Ctx) []observabilityx.Attribute {
	return []observabilityx.Attribute{
		observabilityx.String("method", c.Method()),
		observabilityx.String("path", c.Path()),
		observabilityx.String("status", strconv.Itoa(c.Response().StatusCode())),
	}
}

func assetDeliveryMetricsAttrs(c fiber.Ctx) []observabilityx.Attribute {
	delivery := getAssetDelivery(c)
	if delivery == "" {
		return nil
	}
	return lo.Concat(requestMetricsAttrs(c), []observabilityx.Attribute{
		observabilityx.String("delivery", delivery),
	})
}

func requestLogMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		startedAt := time.Now()
		err := c.Next()
		logRequest(logger, c, startedAt)
		if err != nil {
			return oops.In("server").Owner("request log middleware").Wrap(err)
		}
		return nil
	}
}

func routePattern(mountPath string) string {
	mountPath = strings.TrimSpace(mountPath)
	if lo.Contains([]string{"", "/"}, mountPath) {
		return "/*"
	}
	return strings.TrimRight(mountPath, "/") + "*"
}

func trimMountPath(requestPath, mountPath string) string {
	mountPath = strings.TrimSpace(mountPath)
	if lo.Contains([]string{"", "/"}, mountPath) {
		return strings.TrimPrefix(requestPath, "/")
	}

	trimmed := strings.TrimPrefix(requestPath, strings.TrimRight(mountPath, "/"))
	return strings.TrimPrefix(trimmed, "/")
}

func parsePositiveInt(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func shouldVaryAccept(sourceMediaType, explicitFormat string) bool {
	if strings.TrimSpace(explicitFormat) != "" {
		return false
	}
	return media.IsImageMediaType(sourceMediaType)
}
