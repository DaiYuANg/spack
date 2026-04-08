// Package task for schedule task
package task

import (
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/go-co-op/gocron/v2"
)

var Module = dix.NewModule("task",
	dix.WithModuleProviders(
		dix.ProviderErr1(newScheduler),
	),
	dix.WithModuleHooks(
		dix.OnStart2(startTaskRuntime),
		dix.OnStop(stopTaskRuntime),
	),
)

func newScheduler(logger *slog.Logger) (gocron.Scheduler, error) {
	scheduler, err := gocron.NewScheduler(
		gocron.WithLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}
	return scheduler, nil
}
