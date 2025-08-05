package metrics

import (
	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"net/http"
)

var Module = fx.Module("metrics",
	fx.Provide(
		newServeMux,
		newRequestTotal,
		newRequestDurationSeconds,
		newActiveRequests,
	),
	fx.Invoke(
		start,
		register,
	),
)

func newServeMux() *http.ServeMux {
	return http.NewServeMux()
}

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

func start(lc fx.Lifecycle, mux *http.ServeMux, cfg *config.Config, logger *zap.SugaredLogger) error {
	err := statsviz.Register(mux)
	if err != nil {
		return err
	}
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				logger.Debugf("metrics server start:%s", "http://localhost:8080/debug/statsviz")
				_ = http.ListenAndServe("localhost:8080", mux)
			}()
		},
		func() {},
	))
	return nil
}
