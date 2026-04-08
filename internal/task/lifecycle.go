package task

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

func startTaskRuntime(_ context.Context, scheduler gocron.Scheduler, logger *slog.Logger) error {
	job, err := scheduler.NewJob(
		gocron.DurationJob(10*time.Minute),
		gocron.NewTask(func() {
			logger.Info("health")
		}),
	)
	if err != nil {
		return fmt.Errorf("create job error %w", err)
	}

	logger.Info("Job created", slog.String("id", job.ID().String()))
	scheduler.Start()
	return nil
}

func stopTaskRuntime(ctx context.Context, scheduler gocron.Scheduler) error {
	return scheduler.ShutdownWithContext(ctx)
}
