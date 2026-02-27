package http

import (
	"github.com/daiyuang/spack/internal/registry"
	"github.com/gofiber/fiber/v3"
)

func registryViewMiddleware(app *fiber.App, reg registry.Registry) {
	app.Get("/registry", func(ctx fiber.Ctx) error {
		jsonStr, err := reg.Json()
		if err != nil {
			return ctx.Status(500).SendString(err.Error())
		}
		ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		return ctx.SendString(jsonStr)
	})
}
