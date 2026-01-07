package processor

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"

	"github.com/chai2010/webp"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

type WebPProcessor struct {
	logger      *slog.Logger
	supportMime []constant.MimeType
}

func NewWebPProcessor(logger *slog.Logger) *WebPProcessor {
	return &WebPProcessor{
		logger: logger,
		supportMime: []constant.MimeType{
			constant.Png,
			constant.Jpg,
			constant.Jpeg,
		},
	}
}

func (p *WebPProcessor) Name() string {
	return "WebPProcessor"
}

func (p *WebPProcessor) Match(obj *scanner.ObjectInfo) bool {
	ok := lo.ContainsBy(p.supportMime, func(mt constant.MimeType) bool {
		return mt == obj.Mimetype
	})

	if ok {
		p.logger.Debug(
			"webp matched",
			"path", obj.Key,
			"mime", obj.Mimetype,
		)
	}
	return ok
}

// Run 只做一件事：origin -> webp variant
func (p *WebPProcessor) Run(ctx Context) (int64, error) {
	if ctx.Open == nil || ctx.EmitVariant == nil {
		return 0, oops.New("Context missing Open or EmitVariant")
	}

	// 打开 origin
	r, err := ctx.Open()
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer r.Close()

	img, _, err := image.Decode(r)
	if err != nil {
		return 0, oops.Wrap(err)
	}

	// 编码为 webp 到内存 buffer
	var buf bytes.Buffer
	if err := webp.Encode(
		&buf,
		img,
		&webp.Options{
			Lossless: true,
			Quality:  80,
		},
	); err != nil {
		return 0, oops.Wrap(err)
	}

	// 构造 variant（不涉及 storage）
	variant := &registry.VariantFileInfo{
		Ext:         ".webp",
		VariantType: constant.VariantWebP,
		Size:        int64(buf.Len()),
		Reader:      bytes.NewReader(buf.Bytes()),
	}

	// 交给 lifecycle / context 决定如何持久化
	if err := ctx.EmitVariant(variant); err != nil {
		return variant.Size, oops.Wrap(err)
	}

	return variant.Size, nil
}
