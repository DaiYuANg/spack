package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
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
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	obs observabilityx.Observability,
) *fiber.App {
	engine := html.NewFileSystem(http.FS(view.View), ".html")
	info, ok := debug.ReadBuildInfo()
	header := lo.Ternary(ok, "X-Spack-"+info.Main.Version, "X-Spack")
	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
		Immutable:         true,
		StreamRequestBody: true,
		ErrorHandler:      errorHandler,
		ServerHeader:      header,
		ReduceMemoryUsage: cfg.Http.LowMemory,
	})

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

	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/catalog", func(c fiber.Ctx) error {
		return c.JSON(cat.Snapshot())
	})

	app.Use(routePattern(cfg.Assets.Path), func(c fiber.Ctx) error {
		requestedFormat := normalizeImageFormat(c.Query("format"))
		result, err := assetResolver.Resolve(resolver.Request{
			Path:           trimMountPath(c.Path(), cfg.Assets.Path),
			Accept:         c.Get(fiber.HeaderAccept),
			AcceptEncoding: c.Get(fiber.HeaderAcceptEncoding),
			Width:          parsePositiveInt(c.Query("w")),
			Format:         requestedFormat,
			RangeRequested: strings.TrimSpace(c.Get(fiber.HeaderRange)) != "",
		})
		if err != nil {
			return fiber.ErrNotFound
		}

		if len(result.PreferredEncodings) > 0 || len(result.PreferredWidths) > 0 || len(result.PreferredFormats) > 0 {
			pipelineSvc.Enqueue(pipeline.Request{
				AssetPath:          result.Asset.Path,
				PreferredEncodings: result.PreferredEncodings,
				PreferredWidths:    result.PreferredWidths,
				PreferredFormats:   result.PreferredFormats,
			})
		}

		body, err := os.ReadFile(result.FilePath)
		if err != nil {
			logger.Error("Read asset failed",
				slog.String("path", result.FilePath),
				slog.String("err", err.Error()),
			)
			return fiber.ErrInternalServerError
		}

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
		if result.Variant != nil {
			pipelineSvc.MarkVariantHit(result.FilePath)
		}

		return c.Send(body)
	})

	return app
}

func metricsMiddleware(obs observabilityx.Observability) fiber.Handler {
	if obs == nil {
		obs = observabilityx.NopWithLogger(nil)
	}

	return func(c fiber.Ctx) error {
		requestPath := c.Path()
		method := c.Method()

		startedAt := time.Now()
		err := c.Next()
		duration := time.Since(startedAt).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())

		attrs := []observabilityx.Attribute{
			observabilityx.String("method", method),
			observabilityx.String("path", requestPath),
			observabilityx.String("status", status),
		}
		obs.AddCounter(context.Background(), "http_requests_total", 1, attrs...)
		obs.RecordHistogram(context.Background(), "http_request_duration_seconds", duration, attrs...)
		return err
	}
}

func requestLogMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		startedAt := time.Now()
		err := c.Next()
		logger.Info("HTTP request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(startedAt)),
			slog.String("request_id", c.GetRespHeader("Request-ID")),
		)
		return err
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

func shouldVaryAccept(sourceMediaType string, explicitFormat string) bool {
	if strings.TrimSpace(explicitFormat) != "" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(sourceMediaType)), "image/")
}
