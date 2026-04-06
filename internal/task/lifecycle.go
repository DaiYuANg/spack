package task

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/go-co-op/gocron/v2"
)

func setup(c *dix.Container, _ dix.Lifecycle) error {
	scheduler := dix.MustResolveAs[gocron.Scheduler](c)
	logger := dix.MustResolveAs[*slog.Logger](c)
	_, err := scheduler.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			func() {
				logger.Info("health")
			},
		),
	)
	if err != nil {
		return fmt.Errorf("create job error %w", err)
	}

	return nil
}
