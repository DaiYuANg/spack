package preprocessor

import (
	"github.com/gabriel-vasile/mimetype"
	"go.uber.org/zap"
)

type compressPreprocessor struct {
	logger *zap.SugaredLogger
}

func (c compressPreprocessor) Name() string {
	//TODO implement me
	panic("implement me")
}

func (c compressPreprocessor) CanProcess(path string, mime *mimetype.MIME) bool {
	return false
}

func (c compressPreprocessor) Process(path string) error {
	//TODO implement me
	panic("implement me")
}

func newCompressPreprocessor(logger *zap.SugaredLogger) *compressPreprocessor {
	return &compressPreprocessor{
		logger: logger,
	}
}
