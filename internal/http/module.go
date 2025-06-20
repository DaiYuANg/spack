package http

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v2"
	"go.uber.org/fx"
	"net/http"
	"sproxy/internal/config"
	"sproxy/view"
)

var Module = fx.Module("http",
	fx.Provide(
		newTemplateEngine,
		newServer,
	),
	middlewareModule,
	fx.Invoke(httpLifecycle),
)

func newTemplateEngine() *html.Engine {
	return html.NewFileSystem(http.FS(view.View), ".html")
}

func newServer(engine *html.Engine, config *config.Config) *fiber.App {
	app := fiber.New(
		fiber.Config{
			Views:             engine,
			PassLocalsToViews: true,
			Immutable:         true,
			StreamRequestBody: true,
			CompressedFileSuffixes: map[string]string{
				".gz":  ".gz",
				".br":  ".br",
				".zip": ".zip",
			},
			ErrorHandler: func(ctx fiber.Ctx, err error) error {
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
			},
		},
	)

	return app
}
