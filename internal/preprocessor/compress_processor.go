package preprocessor

import (
	"compress/gzip"
	"github.com/andybalholm/brotli"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/pkg"
	"github.com/klauspost/compress/zstd"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
)

type compressPreprocessor struct {
	logger      *zap.SugaredLogger
	supportMime []constant.MimeType
	pool        *ants.Pool
}

func (c *compressPreprocessor) Name() string {
	return "compress"
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
	c.logger.Debugf("compress process %s", path)
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
		err := c.pool.Submit(func() {
			out := filepath.Join(base, hash+job.ext)
			if _, err := os.Stat(out); os.IsNotExist(err) {
				if err := job.compress(path, out); err != nil {
					c.logger.Warnf("%s compress error: %v", job.ext, err)
				} else {
					c.logger.Debugf("%s created %s", job.ext, out)
				}
			}
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func compressZstd(src, dst string, level int) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		_ = in.Close()
	}(in)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return err
	}
	defer func(enc *zstd.Encoder) {
		_ = enc.Close()
	}(enc)

	_, err = io.Copy(enc, in)
	return err
}

func compressGzip(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	defer gw.Close()

	if err != nil {
		return err
	}

	_, err = io.Copy(gw, in)
	return err
}

func compressBrotli(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	bw := brotli.NewWriterLevel(out, brotli.BestCompression)
	defer bw.Close()

	_, err = io.Copy(bw, in)
	return err
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
