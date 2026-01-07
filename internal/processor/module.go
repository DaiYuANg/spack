package processor

import "go.uber.org/fx"

var Module = fx.Module("processor",
	fx.Provide(
		processorAnnotation(
			NewOriginProcessor,
		),
		processorAnnotation(
			NewGzipVariantProcessor,
		),
		processorAnnotation(
			NewBrotliVariantProcessor,
		),
		processorAnnotation(
			NewZstdVariantProcessor,
		),
		processorAnnotation(
			NewWebPProcessor,
		),
	),
)
