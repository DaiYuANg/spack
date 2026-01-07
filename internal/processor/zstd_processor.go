package processor

import (
	"bytes"
	"io"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/klauspost/compress/zstd"
	"github.com/samber/oops"
)

// ZstdVariantProcessor 为所有文件生成 zst 变体
type ZstdVariantProcessor struct{}

func NewZstdVariantProcessor() *ZstdVariantProcessor {
	return &ZstdVariantProcessor{}
}

func (p *ZstdVariantProcessor) Name() string {
	return "ZstdVariantProcessor"
}

// 对所有文件都处理
func (p *ZstdVariantProcessor) Match(_ *scanner.ObjectInfo) bool {
	return true
}

func (p *ZstdVariantProcessor) Run(ctx Context) (int64, error) {
	if ctx.Open == nil || ctx.EmitVariant == nil {
		return 0, oops.New("Context missing Open or EmitVariant")
	}

	r, err := ctx.Open()
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer r.Close()

	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer zw.Close()

	size, err := io.Copy(zw, r)
	if err != nil {
		return size, oops.Wrap(err)
	}

	variant := &registry.VariantFileInfo{
		Ext:         ".zst",
		VariantType: constant.VariantZstd,
		Size:        int64(buf.Len()),
		Reader:      r,
	}

	if err := ctx.EmitVariant(variant); err != nil {
		return size, oops.Wrap(err)
	}

	return size, nil
}
