package http

import (
	"strconv"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/lifecycle"
	"github.com/gofiber/fiber/v2"
)

func prometheusMiddleware(dep lifecycle.IndicatorDependency) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start).Seconds()
		dep.HttpRequestDurationSeconds.WithLabelValues(c.Method(), c.Path()).Observe(duration)
		dep.HttpRequestsTotal.WithLabelValues(c.Method(), c.Path(), strconv.Itoa(c.Response().StatusCode())).Inc()

		return err
	}
}

func registerPrometheus(app *fiber.App, dep lifecycle.IndicatorDependency, metricsCfg *config.Metrics) {
	app.Use(prometheusMiddleware(dep))
	prometheus := fiberprometheus.New("my-service-name")
	prometheus.RegisterAt(app, metricsCfg.Prefix)
	prometheus.SetSkipPaths([]string{"/ping"})            // Optional: Remove some paths from metrics
	prometheus.SetIgnoreStatusCodes([]int{401, 403, 404}) // Optional: Skip metrics for these status codes
	app.Use(prometheus.Middleware)
}
