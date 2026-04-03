package event

import (
	"context"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/eventx"
)

var Module = dix.NewModule("event",
	dix.WithModuleProviders(
		dix.Provider0(func() eventx.BusRuntime {
			return eventx.New()
		}),
	),
	dix.WithModuleHooks(
		dix.OnStop(func(ctx context.Context, bus eventx.BusRuntime) error {
			return bus.Close()
		}),
	),
)
