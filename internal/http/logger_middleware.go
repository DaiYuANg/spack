package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	slogfiber "github.com/samber/slog-fiber"
)

func loggerMiddleware(app *fiber.App, slogger *slog.Logger) {
	config := slogfiber.Config{
		WithSpanID:  true,
		WithTraceID: true,
	}
	app.Use(slogfiber.NewWithConfig(slogger, config))
}
