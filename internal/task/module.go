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
		dix.Provider2(newTaskRuntime),
	),
	dix.WithModuleHooks(
		dix.OnStart(startTaskRuntime),
		dix.OnStop(stopTaskRuntime),
	),
)

type taskRuntime struct {
	scheduler gocron.Scheduler
	logger    *slog.Logger
}

func newTaskRuntime(scheduler gocron.Scheduler, logger *slog.Logger) *taskRuntime {
	return &taskRuntime{
		scheduler: scheduler,
		logger:    logger,
	}
}
