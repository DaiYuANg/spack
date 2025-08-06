package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func loggerMiddleware(app *fiber.App) {
	app.Use(
		logger.New(
			logger.Config{
				Format: "\"${ip} - - [${time}] \"${method} ${url} ${protocol}\" ${status} ${bytesSent} \"${referer}\" \"${ua}\"\\n\"\n",
			}),
	)
}
