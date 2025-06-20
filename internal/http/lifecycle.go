package http

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"sproxy/internal/config"
	"strconv"
)

type LifecycleDependency struct {
	fx.In
	Lc     fx.Lifecycle
	App    *fiber.App
	Config *config.Config
}

func httpLifecycle(dep LifecycleDependency) {
	lc, app, cfg := dep.Lc, dep.App, dep.Config
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				err := app.Listen(":" + strconv.Itoa(cfg.Http.Port))
				if err != nil {
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
