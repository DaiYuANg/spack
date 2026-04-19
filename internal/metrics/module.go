// Package metrics exposes metrics-related services.
package metrics

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/observabilityx"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

func NewModule(observer *DeferredObserver) dix.Module {
	if observer == nil {
		observer = NewDeferredObserver()
	}
	return dix.NewModule("metrics",
		dix.WithModuleProviders(
			dix.Value(observer),
			dix.Provider2(func(logger *slog.Logger, observer *DeferredObserver) *obsprom.Adapter {
				adapter := NewAdapter(logger)
				if observer != nil {
					observer.Attach(adapter)
				}
				return adapter
			}),
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
