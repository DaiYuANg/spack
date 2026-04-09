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
		return sendErrorResponse(ctx, fiber.StatusNotFound, "Not found")
	default:
		return sendErrorResponse(ctx, code, err.Error())
	}
}

func sendErrorResponse(ctx fiber.Ctx, code int, body string) error {
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	if err := ctx.Status(code).SendString(body); err != nil {
		return fmt.Errorf("send %d response body: %w", code, err)
	}
	return nil
}
