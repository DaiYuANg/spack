package source

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
	"github.com/samber/do/v2"
)

var Module = dix.NewModule("source",
	dix.WithModuleProviders(
		dix.RawProviderWithMetadata(registerSourceProvider, dix.ProviderMetadata{
			Label:  "SourceProvider",
			Output: dix.TypedService[Source](),
			Dependencies: dix.ServiceRefs(
				dix.TypedService[*config.Assets](),
				dix.TypedService[*slog.Logger](),
			),
			Raw: true,
		}),
	),
)

func registerSourceProvider(c *dix.Container) {
	do.ProvideNamed(c.Raw(), dix.TypedService[Source]().Name, func(i do.Injector) (Source, error) {
		cfg, err := do.InvokeNamed[*config.Assets](i, dix.TypedService[*config.Assets]().Name)
		if err != nil {
			return nil, err
		}
		logger, err := do.InvokeNamed[*slog.Logger](i, dix.TypedService[*slog.Logger]().Name)
		if err != nil {
			return nil, err
		}
		return newSourceFromConfig(cfg, logger)
	})
}
