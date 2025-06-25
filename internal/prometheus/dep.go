package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

type IndicatorDependency struct {
	fx.In
	HttpRequestsTotal          *prometheus.CounterVec
	HttpRequestDurationSeconds *prometheus.HistogramVec
	ActiveRequests             *prometheus.GaugeVec
}
