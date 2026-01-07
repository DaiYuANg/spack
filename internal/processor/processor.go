package processor

import (
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

type Context struct {
	Obj  *scanner.ObjectInfo
	Hash string

	Registry registry.Writer

	Open func() (io.ReadCloser, error)

	EmitVariant func(v *registry.VariantFileInfo) error
}

type Processor interface {
	Name() string

	// 是否处理该 original
	Match(o *scanner.ObjectInfo) bool

	// 真正执行
	Run(
		ctx Context,
	) (size int64, err error)
}
