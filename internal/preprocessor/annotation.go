package preprocessor

import (
	"go.uber.org/fx"
)

func processorAnnotation(t any) interface{} {
	return fx.Annotate(
		t,
		fx.ResultTags(`group:"preprocessor"`),
		fx.As(new(Preprocessor)),
	)
}
