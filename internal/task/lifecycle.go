package task

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

func startTaskRuntime(_ context.Context, runtime *taskRuntime) error {
	job, err := runtime.scheduler.NewJob(
		gocron.DurationJob(10*time.Minute),
		gocron.NewTask(func() {
			runtime.logger.Info("health")
		}),
	)
	if err != nil {
		return fmt.Errorf("create job error %w", err)
	}

	runtime.logger.Info("Job created", slog.String("id", job.ID().String()))
	runtime.scheduler.Start()
	return nil
}

func stopTaskRuntime(ctx context.Context, runtime *taskRuntime) error {
	return runtime.scheduler.ShutdownWithContext(ctx)
}
