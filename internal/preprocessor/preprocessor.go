package preprocessor

import (
	"io"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
)

type Preprocessor interface {
	Name() string

	// 是否关心这个对象（按 ext / mime / size / metadata）
	Match(obj *scanner.ObjectInfo) bool

	// 执行处理
	Process(ctx Context) error
}

type Context struct {
	Obj  *scanner.ObjectInfo
	Hash string

	Registry registry.Writer

	// 只读能力
	Open func() (io.ReadCloser, error)

	// 辅助能力（可选）
	EmitVariant func(v *registry.VariantFileInfo) error
}
