package lifecycle

import "go.uber.org/fx"

var Module = fx.Module("lifecycle",
	fx.Invoke(
		startup,
	),
)
