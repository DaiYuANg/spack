package processor

import (
	"bytes"
	"io"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/klauspost/compress/gzip"
	"github.com/samber/oops"
)

// GzipVariantProcessor 为所有文件生成 gzip 变体
type GzipVariantProcessor struct{}

// NewGzipVariantProcessor 构造
func NewGzipVariantProcessor() *GzipVariantProcessor {
	return &GzipVariantProcessor{}
}

// Name 返回唯一标识
func (p *GzipVariantProcessor) Name() string {
	return "GzipVariantProcessor"
}

// Match 对所有文件都处理
func (p *GzipVariantProcessor) Match(_ *scanner.ObjectInfo) bool {
	return true
}

// Run 读取原始文件，生成 gzip，并通过 EmitVariant 注册
func (p *GzipVariantProcessor) Run(ctx Context) (int64, error) {
	if ctx.Open == nil || ctx.EmitVariant == nil {
		return 0, oops.New("Context missing Open or EmitVariant")
	}

	r, err := ctx.Open()
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer r.Close()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	// copy 原始文件到 gzip writer
	size, err := io.Copy(gw, r)
	if err != nil {
		return size, oops.Wrap(err)
	}

	if err := gw.Close(); err != nil {
		return size, oops.Wrap(err)
	}

	// 构造 VariantFileInfo
	variant := &registry.VariantFileInfo{
		Path:        ctx.Obj.Key + ".gz",  // 变体路径
		Ext:         ".gz",                // 后缀
		VariantType: constant.VariantGzip, // 类型
		Size:        int64(buf.Len()),     // gzip 后大小
	}

	// 注册到 registry
	if err := ctx.EmitVariant(variant); err != nil {
		return size, oops.Wrap(err)
	}

	return size, nil
}
