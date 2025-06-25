package http

import (
	"github.com/gofiber/contrib/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/etag"
	expvarmw "github.com/gofiber/fiber/v3/middleware/expvar"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"sproxy/internal/config"
	"time"
)

var middlewareModule = fx.Module("middleware",
	fx.Invoke(
		compressMiddleware,
		etagMiddleware,
		loggerMiddleware,
		requestIdMiddleware,
		helmetMiddleware,
		limiterMiddleware,
		faviconMiddleware,
		monitorMiddleware,
		registerPrometheus,
		spaMiddleware,
		proxyMiddleware,
		debugMiddleware,
		setupPreload,
	),
)

func compressMiddleware(app *fiber.App) {
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
}

func etagMiddleware(app *fiber.App) {
	app.Use(etag.New())
}

func loggerMiddleware(app *fiber.App) {
	app.Use(logger.New(logger.Config{
		Format: "\"${ip} - - [${time}] \"${method} ${url} ${protocol}\" ${status} ${bytesSent} \"${referer}\" \"${ua}\"\\n\"",
	}))
}

func requestIdMiddleware(app *fiber.App) {
	app.Use(requestid.New())
}

func helmetMiddleware(app *fiber.App) {
	app.Use(helmet.New())
}

func limiterMiddleware(app *fiber.App, cfg *config.Config) {
	if !cfg.Limit.Enable {
		return
	}
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Second,
	}))
}

func faviconMiddleware(app *fiber.App) {
	app.Use(favicon.New())
}

func debugMiddleware(app *fiber.App, config *config.Config) {
	if lo.IsEmpty(config.Debug.Prefix) {
		return
	}
	app.Use(expvarmw.New())
	app.Use(pprof.New(pprof.Config{Prefix: config.Debug.Prefix}))
}

func monitorMiddleware(app *fiber.App) {
	app.Get("/monitor", monitor.New())
}
