package http

import (
	"github.com/daiyuang/spack/internal/registry"
	"github.com/gofiber/fiber/v2"
)

func registryViewMiddleware(app *fiber.App, registry registry.Registry) {
	app.Get("/registry", func(ctx *fiber.Ctx) error {
		view := registry.ViewData()
		return ctx.Render("registry", fiber.Map{
			"Originals": view.Originals,
			"Variants":  view.Variants,
		})
	})
}
