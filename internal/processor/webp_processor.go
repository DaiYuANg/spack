package processor

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

type WebPProcessor struct {
	logger      *slog.Logger
	supportMime []constant.MimeType
}

// NewWebPProcessor 构造
func NewWebPProcessor(logger *slog.Logger) *WebPProcessor {
	return &WebPProcessor{
		logger:      logger,
		supportMime: []constant.MimeType{constant.Png, constant.Jpg, constant.Jpeg},
	}
}

// Name 返回唯一标识
func (p *WebPProcessor) Name() string {
	return "WebPProcessor"
}

// Match 根据 MIME 判断是否需要处理
func (p *WebPProcessor) Match(obj *scanner.ObjectInfo) bool {
	mimeType := obj.Mimetype
	p.logger.Debug("File mimetype detected",
		slog.String("path", obj.Key),
		slog.String("mimetype", mimeType),
	)
	ok := lo.ContainsBy(p.supportMime, func(mt constant.MimeType) bool {
		return string(mt) == mimeType
	})
	if ok {
		p.logger.Debug("webp: matched mime=%s for path=%s", mimeType, obj.Key)
	}
	return ok
}

// Run 执行 WebP 转换
func (p *WebPProcessor) Run(ctx Context) (int64, error) {
	if ctx.Open == nil || ctx.EmitVariant == nil {
		return 0, oops.New("Context missing Open or EmitVariant")
	}

	// 生成缓存目录
	cacheDir, err := getCacheDir()
	if err != nil {
		return 0, oops.Wrap(err)
	}

	targetPath := filepath.Join(cacheDir, ctx.Hash+".webp")

	// 如果已存在缓存文件，直接注册
	if _, err := os.Stat(targetPath); err == nil {
		stat, err := os.Stat(targetPath)
		if err != nil {
			return 0, oops.Wrap(err)
		}
		vinfo := &registry.VariantFileInfo{
			Path:        targetPath,
			Ext:         ".webp",
			VariantType: constant.VariantWebP,
			Size:        stat.Size(),
		}
		if err := ctx.EmitVariant(vinfo); err != nil {
			return 0, oops.Wrap(err)
		}
		p.logger.Debug("webp: cached variant registered %s", targetPath)
		return stat.Size(), nil
	}

	// 打开原文件
	r, err := ctx.Open()
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer r.Close()

	img, _, err := image.Decode(r)
	if err != nil {
		return 0, oops.Wrap(err)
	}

	outFile, err := os.Create(targetPath)
	if err != nil {
		return 0, oops.Wrap(err)
	}
	defer outFile.Close()

	if err := webp.Encode(outFile, img, &webp.Options{Lossless: true, Quality: 80}); err != nil {
		return 0, oops.Wrap(err)
	}

	stat, err := os.Stat(targetPath)
	if err != nil {
		return 0, oops.Wrap(err)
	}

	vinfo := &registry.VariantFileInfo{
		Path:        targetPath,
		Ext:         ".webp",
		VariantType: constant.VariantWebP,
		Size:        stat.Size(),
	}
	if err := ctx.EmitVariant(vinfo); err != nil {
		return stat.Size(), oops.Wrap(err)
	}

	p.logger.Debug("webp: generated and registered %s", targetPath)
	return stat.Size(), nil
}

// getCacheDir 返回临时缓存目录
func getCacheDir() (string, error) {
	version := pkg.GetVersionFromBuildInfo()
	basePath := filepath.Join(os.TempDir(), version)
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return "", err
	}
	return basePath, nil
}
