package server

import "github.com/prometheus/client_golang/prometheus"

type RuntimeMetrics struct {
	RequestsInFlight prometheus.Gauge
}

func NewRuntimeMetrics() *RuntimeMetrics {
	return &RuntimeMetrics{
		RequestsInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "spack_http_requests_in_flight",
			Help: "Current number of HTTP requests being processed by the server",
		}),
	}
}

func (m *RuntimeMetrics) Collectors() []prometheus.Collector {
	if m == nil {
		return nil
	}
	return []prometheus.Collector{
		m.RequestsInFlight,
	}
}

func (m *RuntimeMetrics) IncRequestsInFlight() {
	if m == nil || m.RequestsInFlight == nil {
		return
	}
	m.RequestsInFlight.Inc()
}

func (m *RuntimeMetrics) DecRequestsInFlight() {
	if m == nil || m.RequestsInFlight == nil {
		return
	}
	m.RequestsInFlight.Dec()
}
