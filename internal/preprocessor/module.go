package preprocessor

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/gabriel-vasile/mimetype"
	lop "github.com/samber/lo/parallel"
	"go.uber.org/fx"
	"go.uber.org/zap"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"path/filepath"
)

var Module = fx.Module("preprocessor",
	fx.Provide(
		processorAnnotation(newWebpPreprocessor),
		processorAnnotation(newCompressPreprocessor),
	),
	fx.Invoke(process),
)

type LifecycleParameter struct {
	fx.In
	Config        *config.Config
	Logger        *zap.SugaredLogger
	Preprocessors []Preprocessor `group:"preprocessor"`
	Lifecycle     fx.Lifecycle
}

func process(param LifecycleParameter) error {
	static := param.Config.Spa.Static
	logger := param.Logger
	preprocessors := param.Preprocessors
	return filepath.Walk(static, func(path string, info fs.FileInfo, err error) error {
		logger.Debugf("walk at %s", path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		mtype, err := mimetype.DetectFile(path)
		logger.Debugf("mimetype %s", mtype)
		lop.ForEach(preprocessors, func(p Preprocessor, index int) {
			if p.CanProcess(path, mtype) {
				logger.Debugf("find preprocessor %s", p.Name())
				err := p.Process(path)
				if err != nil {
					logger.Warnf("preprocessor error %e", err)
				}
			}
		})
		return nil
	})
}
