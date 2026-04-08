// Package metrics exposes metrics-related services.
package metrics

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/observabilityx"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

func NewModule(adapter *obsprom.Adapter) dix.Module {
	if adapter != nil {
		return dix.NewModule("metrics",
			dix.WithModuleProviders(
				dix.Provider0(func() *obsprom.Adapter { return adapter }),
				dix.Provider0(func() observabilityx.Observability { return adapter }),
			),
		)
	}

	return dix.NewModule("metrics",
		dix.WithModuleProviders(
			dix.Provider1(NewAdapter),
			dix.Provider1(func(adapter *obsprom.Adapter) observabilityx.Observability {
				return adapter
			}),
		),
	)
}

func NewAdapter(logger *slog.Logger) *obsprom.Adapter {
	return obsprom.New(
		obsprom.WithNamespace("spack"),
		obsprom.WithLogger(logger),
	)
}
