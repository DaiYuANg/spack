package lifecycle

import (
	"log/slog"
	"net/http"

	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

type IndicatorDependency struct {
	fx.In
	HttpRequestsTotal          *prometheus.CounterVec
	HttpRequestDurationSeconds *prometheus.HistogramVec
	ActiveRequests             *prometheus.GaugeVec
}

func register(dep IndicatorDependency) {
	prometheus.MustRegister(
		dep.HttpRequestsTotal,
		dep.HttpRequestDurationSeconds,
		dep.ActiveRequests,
	)
}

func start(lc fx.Lifecycle, mux *http.ServeMux, cfg *config.Config, logger *slog.Logger) error {
	if !cfg.Debug.Enable {
		return nil
	}
	err := statsviz.Register(mux)
	if err != nil {
		return err
	}
	lc.Append(fx.StartHook(
		func() {
			go func() {
				logger.Info("Metrics server start:%s", slog.String("test", "http://localhost:8080/debug/statsviz"))
				_ = http.ListenAndServe("localhost:8080", mux)
			}()
		},
	))
	return nil
}
