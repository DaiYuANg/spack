package http

import (
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

func loggerMiddleware(app *fiber.App, zapLogger *zap.Logger) {
	app.Use(fiberzap.New(fiberzap.Config{
		Logger: zapLogger,
	}))
	app.Use(
		logger.New(
			logger.Config{
				Format:        "\"${ip} - ${locals:requestid} - [${time}] \"${method} ${url} ${protocol}\" ${status} ${bytesSent} \"${referer}\" \"${ua}\"\"\n",
				DisableColors: false,
			}),
	)
}
