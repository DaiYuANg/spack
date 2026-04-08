package task

import (
	"context"

	"github.com/go-co-op/gocron/v2"
)

func startTaskRuntime(_ context.Context, scheduler gocron.Scheduler, runtime *sourceRescanRuntime) error {
	return startScheduledTasks(scheduler, runtime)
}

func stopTaskRuntime(ctx context.Context, scheduler gocron.Scheduler) error {
	return scheduler.ShutdownWithContext(ctx)
}
