package processor

import (
	"bytes"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	scanner "github.com/daiyuang/spack/internal/scanner"
	"github.com/samber/oops"
)

// BrotliVariantProcessor 为所有文件生成 br 变体
type BrotliVariantProcessor struct{}

func NewBrotliVariantProcessor() *BrotliVariantProcessor {
	return &BrotliVariantProcessor{}
}

func (p *BrotliVariantProcessor) Name() string {
	return "BrotliVariantProcessor"
}

// 对所有文件都处理
func (p *BrotliVariantProcessor) Match(_ *scanner.ObjectInfo) bool {
	return true
}

func (p *BrotliVariantProcessor) Run(ctx Context) (int64, error) {
	if ctx.Open == nil || ctx.EmitVariant == nil {
		return 0, oops.New("Context missing Open or EmitVariant")
	}

	r, err := ctx.Open()
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer r.Close()

	var buf bytes.Buffer
	bw := brotli.NewWriterLevel(&buf, brotli.BestCompression)

	size, err := io.Copy(bw, r)
	if err != nil {
		return size, oops.Wrap(err)
	}

	if err := bw.Close(); err != nil {
		return size, oops.Wrap(err)
	}

	variant := &registry.VariantFileInfo{
		Path:        ctx.Obj.Key + ".br",
		Ext:         ".br",
		VariantType: constant.VariantBrotli,
		Size:        int64(buf.Len()),
	}

	if err := ctx.EmitVariant(variant); err != nil {
		return size, oops.Wrap(err)
	}

	return size, nil
}
