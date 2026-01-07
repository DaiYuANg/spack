package processor

import (
	"io"
	"path/filepath"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/samber/oops"
)

// OriginProcessor 用于收集原始文件信息并写入 registry
type OriginProcessor struct{}

// NewOriginProcessor 构造函数
func NewOriginProcessor() *OriginProcessor {
	return &OriginProcessor{}
}

// Name 唯一标识
func (p *OriginProcessor) Name() string {
	return "OriginProcessor"
}

// Match 始终处理所有 original
func (p *OriginProcessor) Match(_ *scanner.ObjectInfo) bool {
	return true
}

// Run 收集 metadata 并注册到 registry
func (p *OriginProcessor) Run(ctx Context) (int64, error) {
	var size int64
	if ctx.Open != nil {
		r, err := ctx.Open()
		if err != nil {
			return 0, oops.Wrap(err)
		}
		defer r.Close()

		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			size += int64(n)
			if err != nil {
				if err == io.EOF {
					break
				}
				return size, oops.Wrap(err)
			}
		}
	}

	// 构造 OriginalFileInfo（注意部分信息可以直接用 scanner 提供的 Context.Obj）
	info := &registry.OriginalFileInfo{
		Path: ctx.Obj.Key,
		Size: ctx.Obj.Size, // 或者用 size
		Hash: ctx.Hash,
		Ext:  filepath.Ext(ctx.Obj.Key),
	}

	// 写入 registry
	if err := ctx.Registry.RegisterOriginal(info); err != nil {
		return size, oops.Wrap(err)
	}

	return size, nil
}
