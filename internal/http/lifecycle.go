package http

import (
	"context"
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type LifecycleDependency struct {
	fx.In
	Lc     fx.Lifecycle
	App    *fiber.App
	Config *config.Config
	Logger *slog.Logger
}

func httpLifecycle(dep LifecycleDependency) {
	lc, app, cfg, log := dep.Lc, dep.App, dep.Config, dep.Logger
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				localAddress := "http://127.0.0.1:" + cfg.Http.GetPort()
				log.Info("Http Listening on %s", localAddress)
				err := app.Listen(
					":" + cfg.Http.GetPort(),
				)
				if err != nil {
					log.Error("spack start fail: %v", err) // 打印原始错误
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
