package preprocessor

import (
	"context"
	"io"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
)

type VariantPlan struct {
	VariantType constant.VariantType
	Ext         string
	Params      map[string]string
}

type Processor interface {
	Name() string

	// 是否处理该 original
	Match(o *registry.OriginalFileInfo) bool

	// 仅声明，不做 IO
	Plan(o *registry.OriginalFileInfo) []VariantPlan

	// 真正执行
	Run(
		ctx context.Context,
		obj *scanner.ObjectInfo,
		original *registry.OriginalFileInfo,
		plan VariantPlan,
		w io.Writer,
	) (size int64, err error)
}
