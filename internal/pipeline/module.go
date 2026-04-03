package pipeline

import "go.uber.org/fx"

var Module = fx.Module("pipeline",
	fx.Provide(
		newMetrics,
		fx.Annotate(
			newImageStage,
			fx.ResultTags(`group:"pipeline_stage"`),
		),
		fx.Annotate(
			newCompressionStage,
			fx.ResultTags(`group:"pipeline_stage"`),
		),
		newService,
	),
)
