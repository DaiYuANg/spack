package processor

import "go.uber.org/fx"

var Module = fx.Module("processor",
	fx.Provide(
		processorAnnotation(
			NewOriginProcessor,
		),
	),
)
