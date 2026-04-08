package workerpool

import (
	"context"
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/panjf2000/ants/v2"
)

var Module = dix.NewModule("workerpool",
	dix.WithModuleProviders(
		dix.Provider1(newSettings),
		dix.ProviderErr1(newPool),
	),
	dix.WithModuleHooks(
		dix.OnStop(func(ctx context.Context, pool *ants.Pool) error {
			if pool == nil {
				return nil
			}
			if err := pool.ReleaseTimeout(releaseTimeout()); err != nil && !errors.Is(err, ants.ErrPoolClosed) {
				return fmt.Errorf("release worker pool: %w", err)
			}
			return nil
		}),
	),
)
