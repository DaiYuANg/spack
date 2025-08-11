package spa

import "go.uber.org/fx"

var Module = fx.Module("spa", fx.Provide(
	NewProcessor,
))
