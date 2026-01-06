package scanner

import (
	"go.uber.org/fx"
)

func processorAnnotation(t any) interface{} {
	return fx.Annotate(
		t,
		fx.ResultTags(`group:"scanner"`),
		fx.As(new(Preprocessor)),
	)
}
