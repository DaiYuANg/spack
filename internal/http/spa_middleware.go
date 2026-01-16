package http

import (
	"log/slog"
	"strings"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/finder"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type SpaMiddlewareDependency struct {
	fx.In
	App               *fiber.App
	Config            *config.Config
	Log               *slog.Logger
	HttpRequestsTotal *prometheus.CounterVec
	Finder            *finder.Finder
}

func spaMiddleware(dep SpaMiddlewareDependency) {
	app, cfg, log, total, f := dep.App, dep.Config, dep.Log, dep.HttpRequestsTotal, dep.Finder

	servePath := strings.TrimSpace(cfg.Spa.Path) + "*"
	app.Use(servePath, func(c *fiber.Ctx) error {
		incr := func(label string) {
			total.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}
		reqPath := strings.TrimPrefix(c.Path(), "/")

		lookup, err := f.Lookup(finder.NewLookupContext(c.Get("Accept-Encoding"), reqPath))
		if err != nil {
			log.Error("Find error", slog.Any("error", oops.Wrap(err)))
			if cfg.Spa.NotFoundFallback {
				result, err := f.Lookup(finder.NewLookupContext("", cfg.Spa.Fallback))
				if err != nil {
					log.Error("Find error", slog.Any("error", oops.Wrap(err)))
					return fiber.ErrNotFound
				}
				c.Set(fiber.HeaderContentType, result.MediaTypeString())
				return c.Send(result.Data)
			}
			return fiber.ErrNotFound
		}

		incr("not_found")

		c.Set(fiber.HeaderContentType, lookup.MediaTypeString())
		return c.Send(lookup.Data)
	})
}
