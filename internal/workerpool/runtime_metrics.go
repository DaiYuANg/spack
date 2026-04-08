package workerpool

import (
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type RuntimeMetrics struct {
	collectors []prometheus.Collector
}

func NewRuntimeMetrics(settings *Settings, pool *ants.Pool) *RuntimeMetrics {
	if settings == nil {
		settings = &Settings{Size: 1}
	}

	return &RuntimeMetrics{
		collectors: []prometheus.Collector{
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name: "spack_workerpool_capacity_current",
				Help: "Current shared worker pool capacity",
			}, func() float64 {
				if pool == nil {
					return float64(settings.Size)
				}
				return float64(pool.Cap())
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name: "spack_workerpool_running_current",
				Help: "Current number of running worker pool goroutines",
			}, func() float64 {
				if pool == nil {
					return 0
				}
				return float64(pool.Running())
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name: "spack_workerpool_waiting_current",
				Help: "Current number of waiting tasks in the shared worker pool",
			}, func() float64 {
				if pool == nil {
					return 0
				}
				return float64(pool.Waiting())
			}),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name: "spack_workerpool_free_current",
				Help: "Current number of free workers in the shared worker pool",
			}, func() float64 {
				if pool == nil {
					return float64(settings.Size)
				}
				return float64(pool.Free())
			}),
		},
	}
}

func (m *RuntimeMetrics) Collectors() []prometheus.Collector {
	if m == nil {
		return nil
	}
	return m.collectors
}
