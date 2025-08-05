package preprocessor

import (
	"github.com/gabriel-vasile/mimetype"
	"go.uber.org/fx"
)

type Preprocessor interface {
	// Name 唯一标识（可用于日志等）
	Name() string

	// CanProcess 是否能处理这个文件（含 mime 检测结构体）
	CanProcess(path string, mime *mimetype.MIME) bool

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
