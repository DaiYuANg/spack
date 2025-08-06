package http

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"go.uber.org/zap"
)

func proxyMiddleware(app *fiber.App, config *config.Config, log *zap.SugaredLogger) {
	if !config.Proxy.Enabled() {
		return
	}

	app.Use(
		config.Proxy.Path+"*",
		func(ctx fiber.Ctx) error {
			log.Debugf("into proxy %s", ctx.OriginalURL())
			ctx.Set(constant.PROXY, ctx.OriginalURL())
			return proxy.Do(ctx, config.Proxy.Target)
		},
	)
}
