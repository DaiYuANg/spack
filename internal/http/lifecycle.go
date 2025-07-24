package http

import (
	"context"
	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
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
				err := app.Listen(
					":"+cfg.Http.GetPort(),
					fiber.ListenConfig{
						DisableStartupMessage: true,
						EnablePrintRoutes:     false,
						EnablePrefork:         false,
						ShutdownTimeout:       1000,
					},
				)
				if err != nil {
					log.Errorf("sproxy start fail: %v", err) // 打印原始错误
					panic(err)
				}
			}()
		},
		func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	),
	)
}
