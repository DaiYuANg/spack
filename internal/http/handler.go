package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

func errorHandler(ctx fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	switch code {
	case 404:
		return ctx.Status(404).Render("404", fiber.Map{
			"Path":    ctx.OriginalURL(),
			"Message": "Not found",
		})
	default:
		return ctx.Status(code).Render("500", fiber.Map{
			"Message": err.Error(),
		})
	}
}
