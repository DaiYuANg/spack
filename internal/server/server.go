package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
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
)

func newServer(
	cfg *config.Config,
	logger *slog.Logger,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	obs observabilityx.Observability,
	bus eventx.BusRuntime,
) *fiber.App {
	app := fiber.New(fiber.Config{
		Views:             html.NewFileSystem(http.FS(view.View), ".html"),
		PassLocalsToViews: true,
		Immutable:         true,
		StreamRequestBody: true,
		ErrorHandler:      errorHandler,
		ServerHeader:      buildServerHeader(),
		ReduceMemoryUsage: cfg.HTTP.LowMemory,
	})
	registerMiddleware(app, cfg, logger, obs)
	registerHealthRoutes(app, cat)
	registerAssetRoute(app, cfg, logger, assetResolver, pipelineSvc, bodyCache, bus)
	return app
}

func buildServerHeader() string {
	info, ok := debug.ReadBuildInfo()
	return lo.Ternary(ok, "X-Spack-"+info.Main.Version, "X-Spack")
}

func registerMiddleware(app *fiber.App, cfg *config.Config, logger *slog.Logger, obs observabilityx.Observability) {
	requestIDConfig := requestid.ConfigDefault
	requestIDConfig.Header = "Request-ID"
	app.Use(requestid.New(requestIDConfig))
	app.Use(etag.New())
	app.Use(helmet.New())
	app.Use(requestLogMiddleware(logger))
	app.Use(metricsMiddleware(obs))

	if cfg.Debug.Enable {
		app.Use(expvarmw.New())
		app.Use(pprof.New(pprof.Config{Prefix: cfg.Debug.PprofPrefix}))
	}

	recoverConfig := recoverer.ConfigDefault
	recoverConfig.EnableStackTrace = true
	app.Use(recoverer.New(recoverConfig))
}

func registerHealthRoutes(app *fiber.App, cat catalog.Catalog) {
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/catalog", func(c fiber.Ctx) error {
		return c.JSON(cat.Snapshot())
	})
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
		requestedFormat := normalizeImageFormat(c.Query("format"))
		request := buildResolverRequest(c, cfg.Assets.Path, requestedFormat)
		result, err := assetResolver.Resolve(request)
		if err != nil {
			return fiber.ErrNotFound
		}

		enqueuePipelineResult(result, pipelineSvc)
		delivery, err := sendResolvedAsset(c, request, result, requestedFormat, logger, bodyCache)
		if err != nil {
			return err
		}
		setAssetDelivery(c, delivery)
		publishVariantServed(c.Context(), result, bus, logger)
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

func applyResolvedHeaders(c fiber.Ctx, result *resolver.Result, requestedFormat string) {
	c.Set(fiber.HeaderContentType, result.MediaType)
	c.Append(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
	if shouldVaryAccept(result.Asset.MediaType, requestedFormat) {
		c.Append(fiber.HeaderVary, fiber.HeaderAccept)
	}
	if result.ETag != "" {
		c.Set(fiber.HeaderETag, result.ETag)
	}
	if result.ContentEncoding != "" {
		c.Set(fiber.HeaderContentEncoding, result.ContentEncoding)
	}
}

func metricsMiddleware(obs observabilityx.Observability) fiber.Handler {
	if obs == nil {
		obs = observabilityx.NopWithLogger(nil)
	}

	return func(c fiber.Ctx) error {
		startedAt := time.Now()
		err := c.Next()
		duration := time.Since(startedAt).Seconds()

		requestAttrs := requestMetricsAttrs(c)
		obs.AddCounter(context.Background(), "http_requests_total", 1, requestAttrs...)
		obs.RecordHistogram(context.Background(), "http_request_duration_seconds", duration, requestAttrs...)

		deliveryAttrs := assetDeliveryMetricsAttrs(c)
		if len(deliveryAttrs) > 0 {
			obs.AddCounter(context.Background(), "http_asset_delivery_total", 1, deliveryAttrs...)
			obs.RecordHistogram(context.Background(), "http_asset_delivery_duration_seconds", duration, deliveryAttrs...)
		}
		if err != nil {
			return fmt.Errorf("run metrics middleware chain: %w", err)
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
	return append(requestMetricsAttrs(c), observabilityx.String("delivery", delivery))
}

func requestLogMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		startedAt := time.Now()
		err := c.Next()
		logRequest(logger, c, startedAt)
		if err != nil {
			return fmt.Errorf("run request log middleware chain: %w", err)
		}
		return nil
	}
}

func routePattern(mountPath string) string {
	mountPath = strings.TrimSpace(mountPath)
	if mountPath == "" || mountPath == "/" {
		return "/*"
	}
	return strings.TrimRight(mountPath, "/") + "*"
}

func trimMountPath(requestPath, mountPath string) string {
	mountPath = strings.TrimSpace(mountPath)
	if mountPath == "" || mountPath == "/" {
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

func normalizeImageFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	default:
		return ""
	}
}

func shouldVaryAccept(sourceMediaType, explicitFormat string) bool {
	if strings.TrimSpace(explicitFormat) != "" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(sourceMediaType)), "image/")
}
