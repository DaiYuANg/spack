package finder

import "go.uber.org/fx"

var Module = fx.Module("finder", fx.Provide(
	newFinder,
))
