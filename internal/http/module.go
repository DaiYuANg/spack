package http

import (
	"net/http"
	"runtime/debug"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/view"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"
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
	header := lo.Ternary(ok, "X-Spack-"+info.Main.Version, "X-Spack")
	app := fiber.New(
		fiber.Config{
			Views:                 engine,
			PassLocalsToViews:     true,
			Immutable:             true,
			StreamRequestBody:     true,
			ErrorHandler:          errorHandler,
			ServerHeader:          header,
			ReduceMemoryUsage:     cfg.Http.LowMemory,
			DisableStartupMessage: true,
			EnablePrintRoutes:     false,
			Prefork:               false,
		},
	)

	return app
}
