package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

var Module = fx.Module("prometheus",
	fx.Provide(
		newRequestTotal,
		newRequestDurationSeconds,
		newActiveRequests,
	),
	fx.Invoke(register),
)

func newRequestTotal() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
}

func newRequestDurationSeconds() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of HTTP request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
}

func newActiveRequests() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_requests",
			Help: "Number of active requests",
		},
		[]string{"method", "path"},
	)
}

func register(dep IndicatorDependency) {
	prometheus.MustRegister(
		dep.HttpRequestsTotal,
		dep.HttpRequestDurationSeconds,
		dep.ActiveRequests,
	)
}
