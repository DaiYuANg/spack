package source

import "go.uber.org/fx"

var Module = fx.Module("source", fx.Provide(
	newLocalFS,
))
