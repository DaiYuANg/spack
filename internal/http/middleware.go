package http

import (
	"runtime/debug"
	"time"

	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/contrib/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/earlydata"
	"github.com/gofiber/fiber/v3/middleware/envvar"
	"github.com/gofiber/fiber/v3/middleware/etag"
	expvarmw "github.com/gofiber/fiber/v3/middleware/expvar"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

var middlewareModule = fx.Module(
	"middleware",
	fx.Invoke(
		requestMetaMiddleware,
		earlydataMiddleware,
		corsMiddleware,
		compressMiddleware,
		envvarMiddleware,
		etagMiddleware,
		loggerMiddleware,
		requestIdMiddleware,
		helmetMiddleware,
		limiterMiddleware,
		faviconMiddleware,
		monitorMiddleware,
		registerPrometheus,
		healthcheckMiddleware,
		cacheMiddleware,
		proxyMiddleware,
		debugMiddleware,
		setupPreload,
		spaMiddleware,
		recoverMiddleware,
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
	if !config.Debug.Enable {
		return
	}
	app.Use(expvarmw.New())
	app.Use(pprof.New(pprof.Config{Prefix: config.Debug.PprofPrefix}))
}

func monitorMiddleware(app *fiber.App) {
	app.Get("/monitor", monitor.New())
}

func cacheMiddleware(app *fiber.App) {
	app.Use(cache.New())
}

func recoverMiddleware(app *fiber.App) {
	cfg := recoverer.ConfigDefault
	cfg.EnableStackTrace = true
	app.Use(recoverer.New(cfg))
}

func healthcheckMiddleware(app *fiber.App) {
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
}

func corsMiddleware(app *fiber.App) {
	app.Use(cors.New())
}

func earlydataMiddleware(app *fiber.App) {
	app.Use(earlydata.New(earlydata.Config{
		Error: fiber.ErrTooEarly,
	}))
}

func envvarMiddleware(app *fiber.App) {
	info, _ := debug.ReadBuildInfo()
	resp := map[string]string{
		"main":     info.Main.Path,
		"version":  info.Main.Version,
		"mod_time": info.Main.Sum,
	}
	lo.ForEach(info.Settings, func(item debug.BuildSetting, index int) {
		resp[item.Key] = item.Value
	})
	app.Use("/expose/envvars", envvar.New(
		envvar.Config{
			ExportVars: resp,
		},
	))
}

func requestMetaMiddleware(app *fiber.App, cfg *config.Config, httpRequestsTotal *prometheus.CounterVec) {
	app.Use(requestMeta(cfg, httpRequestsTotal))
}
