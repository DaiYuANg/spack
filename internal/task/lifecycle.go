package task

import (
	"context"
	"fmt"

	"github.com/go-co-op/gocron/v2"
)

func startTaskRuntime(ctx context.Context, scheduler gocron.Scheduler, runtime *sourceRescanRuntime) error {
	return startScheduledTasks(context.WithoutCancel(ctx), scheduler, runtime)
}

func stopTaskRuntime(ctx context.Context, scheduler gocron.Scheduler) error {
	if err := scheduler.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("shutdown task scheduler: %w", err)
	}
	return nil
}
