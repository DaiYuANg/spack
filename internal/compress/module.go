package compress

import "go.uber.org/fx"

var Module = fx.Module("compress",
	fx.Provide(newService),
)
