// Package metrics exposes metrics-related services.
package metrics

import (
	"errors"
	"log/slog"

	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/observabilityx"
	obsprom "github.com/arcgolabs/observabilityx/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/oops"
)

var Module = dix.NewModule("metrics",
	dix.WithModuleProviders(
		dix.Provider1(NewAdapter),
		dix.Provider1(func(adapter *obsprom.Adapter) observabilityx.Observability {
			return adapter
		}),
		dix.Provider1(func(obs observabilityx.Observability) collectionx.List[dix.Observer] {
			return collectionx.NewList[dix.Observer](NewObserver(obs))
		}),
	),
)

func NewAdapter(logger *slog.Logger) *obsprom.Adapter {
	return obsprom.New(
		obsprom.WithNamespace("spack"),
		obsprom.WithLogger(logger),
		obsprom.WithRegisterer(prometheusAlreadyRegisteredCompat{Registerer: prometheus.DefaultRegisterer}),
	)
}

type prometheusAlreadyRegisteredCompat struct {
	prometheus.Registerer
}

func (r prometheusAlreadyRegisteredCompat) Register(collector prometheus.Collector) error {
	err := r.Registerer.Register(collector)
	if err == nil {
		return nil
	}

	if alreadyRegistered, ok := errors.AsType[prometheus.AlreadyRegisteredError](err); ok {
		return &alreadyRegistered
	}
	return oops.In("metrics").Owner("prometheus registerer").Wrap(err)
}
