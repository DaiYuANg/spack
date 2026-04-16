package workerpool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/oops"
)

var Module = dix.NewModule("workerpool",
	dix.WithModuleProviders(
		dix.Provider1(newSettings),
		dix.ProviderErr1(newPool),
		dix.Provider2(NewRuntimeMetrics),
	),
	dix.WithModuleHooks(
		dix.OnStop2(func(ctx context.Context, pool *ants.Pool, logger *slog.Logger) error {
			if pool == nil {
				return nil
			}
			logger.Info("close ant pool")
			if err := pool.ReleaseTimeout(releaseTimeout()); err != nil && !errors.Is(err, ants.ErrPoolClosed) {
				return oops.In("ants pool").Wrap(fmt.Errorf("release worker pool: %w", err))
			}
			return nil
		}),
	),
)
