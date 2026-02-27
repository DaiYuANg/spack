package lifecycle

import (
	"context"
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

type Dependency struct {
	fx.In
	Lc     fx.Lifecycle
	App    *fiber.App
	Config *config.Config
	Logger *slog.Logger
}

func httpLifecycle(dep Dependency) {
	lc, app, cfg, log := dep.Lc, dep.App, dep.Config, dep.Logger
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				localAddress := "http://127.0.0.1:" + cfg.Http.GetPort()
				log.Info("Http Listening on %s", slog.String("address", localAddress))
				log.Info("Registry data on %s", slog.String("address", localAddress+"/registry"))
				err := app.Listen(
					":" + cfg.Http.GetPort(),
				)
				if err != nil {
					log.Error("spack start fail: %v", slog.String("err", err.Error())) // 打印原始错误
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
