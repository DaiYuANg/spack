package catalog

import "go.uber.org/fx"

var Module = fx.Module("catalog", fx.Provide(
	fx.Annotate(
		NewInMemoryCatalog,
		fx.As(new(Catalog)),
	),
))
