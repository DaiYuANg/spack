package processor

import (
	"go.uber.org/fx"
)

func processorAnnotation(t any) interface{} {
	return fx.Annotate(
		t,
		fx.ResultTags(`group:"processor"`),
		fx.As(new(Processor)),
	)
}
