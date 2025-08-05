package preprocessor

import (
	"github.com/daiyuang/spack/internal/constant"
	"github.com/gabriel-vasile/mimetype"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type compressPreprocessor struct {
	logger      *zap.SugaredLogger
	supportMime []constant.MimeType
}

func (c *compressPreprocessor) Name() string {
	return "compress"
}

func (c *compressPreprocessor) CanProcess(path string, mime *mimetype.MIME) bool {
	if mime == nil {
		return false
	}

	ok := lo.ContainsBy(c.supportMime, func(mt constant.MimeType) bool {
		return mime.Is(string(mt))
	})

	if ok {
		c.logger.Debugf("webp: matched mime=%s for path=%s", mime.String(), path)
	}

	return ok
}

func (c *compressPreprocessor) Process(path string) error {
	//TODO implement me
	panic("implement me")
}

func newCompressPreprocessor(logger *zap.SugaredLogger) *compressPreprocessor {
	return &compressPreprocessor{
		logger: logger,
	}
}
