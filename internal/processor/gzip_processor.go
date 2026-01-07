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

func (p *GzipVariantProcessor) Name() string {
	return "GzipVariantProcessor"
}

// 对所有文件生效
func (p *GzipVariantProcessor) Match(_ *scanner.ObjectInfo) bool {
	return true
}

// Run 只负责：origin -> gzip variant
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

	// gzip 压缩
	if _, err := io.Copy(gw, r); err != nil {
		return 0, oops.Wrap(err)
	}
	if err := gw.Close(); err != nil {
		return 0, oops.Wrap(err)
	}

	// 构造 variant（不涉及 storage）
	variant := &registry.VariantFileInfo{
		Ext:         ".gz",
		VariantType: constant.VariantGzip,
		Size:        int64(buf.Len()),
		Reader:      bytes.NewReader(buf.Bytes()),
	}

	// lifecycle/context 决定如何落 storage、如何分配 key
	if err := ctx.EmitVariant(variant); err != nil {
		return variant.Size, oops.Wrap(err)
	}

	return variant.Size, nil
}
