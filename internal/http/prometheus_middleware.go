package http

import (
	"strconv"
	"time"

	p "github.com/daiyuang/spack/internal/metrics"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func prometheusMiddleware(dep p.IndicatorDependency) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start).Seconds()
		dep.HttpRequestDurationSeconds.WithLabelValues(c.Method(), c.Path()).Observe(duration)
		dep.HttpRequestsTotal.WithLabelValues(c.Method(), c.Path(), strconv.Itoa(c.Response().StatusCode())).Inc()

		return err
	}
}

func registerPrometheus(app *fiber.App, dep p.IndicatorDependency) {
	app.Get("/prometheus", adaptor.HTTPHandler(promhttp.Handler()))
	app.Use(prometheusMiddleware(dep))
}
