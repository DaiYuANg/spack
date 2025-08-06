package preprocessor

import (
	"go.uber.org/fx"
)

var Module = fx.Module("preprocessor",
	fx.Provide(
		processorAnnotation(newWebpPreprocessor),
		processorAnnotation(newCompressPreprocessor),
	),
	fx.Invoke(preprocess),
)
