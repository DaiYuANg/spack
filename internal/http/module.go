package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v2"
	"go.uber.org/fx"
	"net/http"
	"sproxy/internal/config"
	"sproxy/internal/http/image"
	"sproxy/view"
)

var Module = fx.Module("http",
	fx.Provide(
		newTemplateEngine,
		newServer,
	),
	middlewareModule,
	image.Module,
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
			ErrorHandler:      errorHandler,
		},
	)

	return app
}
