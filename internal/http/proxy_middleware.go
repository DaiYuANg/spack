package http

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
)

func proxyMiddleware(app *fiber.App, config *config.Config, log *slog.Logger) {
	if !config.Proxy.Enabled() {
		return
	}

	app.Use(
		config.Proxy.Path+"*",
		func(ctx *fiber.Ctx) error {
			log.Debug("into proxy", slog.String("url", ctx.OriginalURL()))
			ctx.Set(constant.PROXY, ctx.OriginalURL())
			return proxy.Do(ctx, config.Proxy.Target)
		},
	)
}
