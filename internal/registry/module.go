package registry

import (
	"go.uber.org/fx"
)

var Module = fx.Module("Registry",
	fx.Provide(
		fx.Annotate(
			NewInMemoryRegistry,
			fx.As(new(Registry)),
		),
	),
)
