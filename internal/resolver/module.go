package resolver

import "go.uber.org/fx"

var Module = fx.Module("resolver", fx.Provide(
	newResolver,
))
