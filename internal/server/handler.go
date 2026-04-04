// Package server exposes the HTTP runtime and middleware stack.
package server

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func errorHandler(ctx fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if fiberErr, ok := errors.AsType[*fiber.Error](err); ok {
		code = fiberErr.Code
	}

	switch code {
	case fiber.StatusNotFound:
		if renderErr := ctx.Status(fiber.StatusNotFound).Render("404", fiber.Map{
			"Path":    ctx.OriginalURL(),
			"Message": "Not found",
		}); renderErr != nil {
			return fmt.Errorf("render 404 page: %w", renderErr)
		}
		return nil
	default:
		if renderErr := ctx.Status(code).Render("500", fiber.Map{
			"Message": err.Error(),
		}); renderErr != nil {
			return fmt.Errorf("render error page: %w", renderErr)
		}
		return nil
	}
}
