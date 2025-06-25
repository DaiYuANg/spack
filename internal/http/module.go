package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"net/http"
	"runtime/debug"
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

func newServer(engine *html.Engine, cfg *config.Config) *fiber.App {
	info, ok := debug.ReadBuildInfo()
	header := lo.Ternary(ok, "X-Sproxy-"+info.Main.Version, "X-Sproxy")
	app := fiber.New(
		fiber.Config{
			Views:             engine,
			PassLocalsToViews: true,
			Immutable:         true,
			StreamRequestBody: true,
			ErrorHandler:      errorHandler,
			ServerHeader:      header,
			ReduceMemoryUsage: cfg.Http.LowMemory,
		},
	)

	return app
}
