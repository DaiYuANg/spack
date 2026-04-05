// Package event wires application event infrastructure.
package event

import (
	"context"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/daiyuang/spack/internal/workerpool"
)

var Module = dix.NewModule("event",
	dix.WithModuleProviders(
		dix.Provider1(func(settings *workerpool.Settings) eventx.BusRuntime {
			return eventx.New(
				eventx.WithParallelDispatch(true),
				eventx.WithAntsPool(settings.Size),
			)
		}),
	),
	dix.WithModuleHooks(
		dix.OnStop(func(ctx context.Context, bus eventx.BusRuntime) error {
			return bus.Close()
		}),
	),
)
