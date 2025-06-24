package image

import (
	"github.com/gofiber/fiber/v3"
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"sproxy/internal/config"
)

var Module = fx.Module("image",
	fx.Invoke(
		prepareScan,
		scan,
		imageMiddleware,
	),
)

type MiddlewareDependency struct {
	fx.In
	App    *fiber.App
	Config *config.Config
	Log    *zap.SugaredLogger
	KvMap  *bbolt.DB
}

func imageMiddleware(dep MiddlewareDependency) {
	app, cfg, log := dep.App, dep.Config, dep.Log
	app.Use(optimizedImageMiddleware(cfg, log))
}
