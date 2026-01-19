package http

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/etag"
	expvarmw "github.com/gofiber/fiber/v2/middleware/expvar"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

var middlewareModule = fx.Module(
	"middleware",
	fx.Invoke(
		requestIdMiddleware,
		requestMetaMiddleware,
		etagMiddleware,
		loggerMiddleware,
		helmetMiddleware,
		registerPrometheus,
		healthcheckMiddleware,
		debugMiddleware,
		registryViewMiddleware,
		assetsMiddleware,
		recoverMiddleware,
	),
)

func etagMiddleware(app *fiber.App) {
	app.Use(etag.New())
}

func requestIdMiddleware(app *fiber.App) {
	cfg := requestid.ConfigDefault
	cfg.Header = "Request-ID"
	app.Use(requestid.New(cfg))
}

func helmetMiddleware(app *fiber.App) {
	app.Use(helmet.New())
}

func debugMiddleware(app *fiber.App, debugCfg *config.Debug) {
	if !debugCfg.Enable {
		return
	}
	app.Use(expvarmw.New())
	app.Use(pprof.New(pprof.Config{Prefix: debugCfg.PprofPrefix}))
}

func recoverMiddleware(app *fiber.App) {
	cfg := recoverer.ConfigDefault
	cfg.EnableStackTrace = true
	app.Use(recoverer.New(cfg))
}

func healthcheckMiddleware(app *fiber.App) {
	app.Get(healthcheck.DefaultLivenessEndpoint, healthcheck.New())
}

func requestMetaMiddleware(app *fiber.App, cfg *config.Config, httpRequestsTotal *prometheus.CounterVec) {
	app.Use(requestMeta(cfg, httpRequestsTotal))
}
