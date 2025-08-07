package preprocessor

import (
	"math"
	"os"
	"path/filepath"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type compressPreprocessor struct {
	logger      *zap.SugaredLogger
	supportMime []constant.MimeType
	pool        *ants.Pool
	registry    registry.Registry
}

func (c *compressPreprocessor) Name() string {
	return "compress"
}

func (c *compressPreprocessor) Order() int {
	return math.MaxInt
}

func (c *compressPreprocessor) CanProcess(info *registry.OriginalFileInfo) bool {
	ok := lo.ContainsBy(c.supportMime, func(mt constant.MimeType) bool {
		return string(mt) == info.Mimetype
	})

	if ok {
		c.logger.Debugf("webp: matched mime=%s for path=%s", info.Mimetype, info.Path)
	}

	return ok
}

func (c *compressPreprocessor) Process(info *registry.OriginalFileInfo) error {
	path := info.Path
	c.logger.Debugf("compress preprocess %s", path)

	ext := filepath.Ext(path) // 如 .html

	// 包含扩展名内容的 hash，可避免 .html 和 .js 相同内容冲突
	hash, err := pkg.FileHashWithExt(path)
	if err != nil {
		c.logger.Errorf("hash failed for %s: %v", path, err)
		return err
	}

	version := pkg.GetVersionFromBuildInfo()
	base := filepath.Join(os.TempDir(), version)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}

	protos := []struct {
		ext      string
		compress func(string, string) error
	}{
		{".gz", compressGzip},
		{".br", compressBrotli},
		{".zst", func(s, d string) error { return compressZstd(s, d, 3) }},
	}

	for _, proto := range protos {
		job := proto

		if err := c.pool.Submit(func() {
			// index.html.gz
			siblingCompressed := path + job.ext

			// hash.html.gz
			out := filepath.Join(base, hash+ext+job.ext)

			// 如果同目录下已有压缩文件，则跳过
			if _, err := os.Stat(siblingCompressed); err == nil {
				c.logger.Debugf("%s exists in source dir, skip compression", siblingCompressed)
				return
			}

			if _, err := os.Stat(out); os.IsNotExist(err) {
				if err := job.compress(path, out); err != nil {
					c.logger.Warnf("%s compress error: %v", job.ext, err)
				} else {
					c.logger.Debugf("%s created %s", job.ext, out)
				}
			}
		}); err != nil {
			return err
		}
	}

	return nil
}

func newCompressPreprocessor(logger *zap.SugaredLogger, pool *ants.Pool, r registry.Registry) *compressPreprocessor {
	return &compressPreprocessor{
		logger:   logger,
		pool:     pool,
		registry: r,
		supportMime: []constant.MimeType{
			constant.Css,
			constant.TextJavascript,
			constant.ApplicationJavascript,
			constant.Html,
			constant.Json,
			constant.Svg,
		},
	}
}
