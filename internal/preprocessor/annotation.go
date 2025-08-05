package preprocessor

import (
	"go.uber.org/fx"
)

type Preprocessor interface {
	// Name 唯一标识（可用于日志等）
	Name() string

	CanProcess(path string, mime string) bool

	// Process 实际执行处理逻辑
	Process(path string) error
}

func processorAnnotation(t any) interface{} {
	return fx.Annotate(
		t,
		fx.ResultTags(`group:"preprocessor"`),
		fx.As(new(Preprocessor)),
	)
}
