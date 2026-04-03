package server

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

func errorHandler(ctx fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
	}

	switch code {
	case fiber.StatusNotFound:
		return ctx.Status(fiber.StatusNotFound).Render("404", fiber.Map{
			"Path":    ctx.OriginalURL(),
			"Message": "Not found",
		})
	default:
		return ctx.Status(code).Render("500", fiber.Map{
			"Message": err.Error(),
		})
	}
}
