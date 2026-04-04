// Package metrics exposes metrics-related services.
package metrics

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/observabilityx"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

var Module = dix.NewModule("metrics",
	dix.WithModuleProviders(
		dix.Provider1(newAdapter),
		dix.Provider1(func(adapter *obsprom.Adapter) observabilityx.Observability {
			return adapter
		}),
	),
)

func newAdapter(logger *slog.Logger) *obsprom.Adapter {
	return obsprom.New(
		obsprom.WithNamespace("spack"),
		obsprom.WithLogger(logger),
	)
}
