package preprocessor

import (
	"math"
	"os"
	"path/filepath"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type compressPreprocessor struct {
	logger      *zap.SugaredLogger
	supportMime []constant.MimeType
	pool        *ants.Pool
}

func (c *compressPreprocessor) Name() string {
	return "compress"
}

func (c *compressPreprocessor) Order() int {
	return math.MaxInt
}

func (c *compressPreprocessor) CanProcess(path string, mimetype string) bool {
	ok := lo.ContainsBy(c.supportMime, func(mt constant.MimeType) bool {
		return string(mt) == mimetype
	})

	if ok {
		c.logger.Debugf("webp: matched mime=%s for path=%s", mimetype, path)
	}

	return ok
}

func (c *compressPreprocessor) Process(path string) error {
	c.logger.Debugf("compress preprocess %s", path)
	version := pkg.GetVersionFromBuildInfo()
	hash, err := pkg.FileHash(path)
	if err != nil {
		c.logger.Errorf("hash failed for %s: %v", path, err)
		return err
	}
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
			// 原始路径同目录下的压缩文件路径
			siblingCompressed := path + job.ext

			// 临时目录中的压缩路径
			out := filepath.Join(base, hash+job.ext)

			// 如果同目录下已有压缩文件，则跳过
			if _, err := os.Stat(siblingCompressed); err == nil {
				c.logger.Debugf("%s exists in source dir, skip compression", siblingCompressed)
				return
			}

			// 如果临时目录中已存在，也跳过
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

func newCompressPreprocessor(logger *zap.SugaredLogger, pool *ants.Pool) *compressPreprocessor {
	return &compressPreprocessor{
		logger: logger,
		pool:   pool,
		supportMime: []constant.MimeType{
			constant.Css,
			constant.TextJavascript,
			constant.ApplicationJavascript,
		},
	}
}
