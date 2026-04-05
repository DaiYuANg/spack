package workerpool

import (
	"context"
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/do/v2"
)

var Module = dix.NewModule("workerpool",
	dix.WithModuleProviders(
		dix.Provider1(newSettings),
		dix.RawProviderWithMetadata(registerPoolProvider, dix.ProviderMetadata{
			Label:  "WorkerPoolProvider",
			Output: dix.TypedService[*ants.Pool](),
			Dependencies: dix.ServiceRefs(
				dix.TypedService[*Settings](),
			),
			Raw: true,
		}),
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

func registerPoolProvider(c *dix.Container) {
	do.ProvideNamed(c.Raw(), dix.TypedService[*ants.Pool]().Name, func(i do.Injector) (*ants.Pool, error) {
		settings, err := do.InvokeNamed[*Settings](i, dix.TypedService[*Settings]().Name)
		if err != nil {
			return nil, err
		}
		return newPool(settings)
	})
}
