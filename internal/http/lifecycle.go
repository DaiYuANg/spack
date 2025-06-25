package http

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"sproxy/internal/config"
)

type LifecycleDependency struct {
	fx.In
	Lc     fx.Lifecycle
	App    *fiber.App
	Config *config.Config
	Logger *zap.SugaredLogger
}

func httpLifecycle(dep LifecycleDependency) {
	lc, app, cfg, log := dep.Lc, dep.App, dep.Config, dep.Logger
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				localAddress := "http://127.0.0.1:" + cfg.Http.GetPort()
				log.Debugf("Http Listening on %s", localAddress)
				lo.Must0(app.Listen(
					":"+cfg.Http.GetPort(),
					fiber.ListenConfig{
						DisableStartupMessage: true,
						EnablePrintRoutes:     true,
						EnablePrefork:         cfg.Http.Prefork,
						ShutdownTimeout:       1000,
					},
				), "sproxy start fail")
			}()
		},
		func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	),
	)
}
