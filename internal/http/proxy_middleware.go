package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"go.uber.org/zap"
	"sproxy/internal/config"
	"sproxy/internal/constant"
)

func proxyMiddleware(app *fiber.App, config *config.Config, log *zap.SugaredLogger) {
	if !config.Proxy.Enabled() {
		return
	}

	app.Use(config.Proxy.Path+"*", func(ctx fiber.Ctx) error {
		log.Debugf("proxy%s", ctx.OriginalURL())
		ctx.Set(constant.PROXY, ctx.OriginalURL())
		return proxy.Do(ctx, config.Proxy.Target)
	})
}
