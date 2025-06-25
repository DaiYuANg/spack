package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"strconv"
	"time"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of HTTP request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	activeRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_requests",
			Help: "Number of active requests",
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDurationSeconds, activeRequests)
}

func prometheusMiddleware(c fiber.Ctx) error {
	start := time.Now()

	// Process the request
	err := c.Next()

	duration := time.Since(start).Seconds()
	httpRequestDurationSeconds.WithLabelValues(c.Method(), c.Path()).Observe(duration)
	httpRequestsTotal.WithLabelValues(c.Method(), c.Path(), strconv.Itoa(c.Response().StatusCode())).Inc()

	return err
}

func registerPrometheus(app *fiber.App) {
	app.Get("/prometheus", adaptor.HTTPHandler(promhttp.Handler()))
	app.Use(prometheusMiddleware)
}
