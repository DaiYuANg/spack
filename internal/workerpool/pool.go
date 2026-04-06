package workerpool

import (
	"fmt"
	"time"

	"github.com/daiyuang/spack/internal/config"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/oops"
)

type Settings struct {
	Size int
}

func newSettings(cfg *config.Async) *Settings {
	return &Settings{
		Size: cfg.NormalizedWorkers(),
	}
}

func newPool(settings *Settings) (*ants.Pool, error) {
	if settings == nil {
		settings = &Settings{Size: 1}
	}

	pool, err := ants.NewPool(settings.Size)
	if err != nil {
		return nil, oops.Wrap(fmt.Errorf("create ants pool: %w", err))
	}
	return pool, nil
}

func releaseTimeout() time.Duration {
	return 3 * time.Second
}
