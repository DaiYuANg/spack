// Package task for schedule task
package task

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/go-co-op/gocron/v2"
)

var Module = dix.NewModule("task",
	dix.WithModuleProviders(
		dix.Provider1(func(logger *slog.Logger) gocron.Scheduler {
			s, err := gocron.NewScheduler(
				gocron.WithLogger(logger),
			)
			if err != nil {
				panic(err)
			}
			return s
		}),
	),
	dix.WithModuleSetups(
		dix.SetupWithMetadata(
			setup, dix.SetupMetadata{
				Label: "taskRuntime",
				Dependencies: dix.ServiceRefs(
					dix.TypedService[gocron.Scheduler](),
					dix.TypedService[*slog.Logger](),
				),
			}),
	),
	dix.WithModuleInvokes(dix.Invoke1[gocron.Scheduler](func(scheduler gocron.Scheduler) {
		scheduler.Start()
	})),
)
