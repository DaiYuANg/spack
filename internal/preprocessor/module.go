package preprocessor

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"

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
	Pool          *ants.Pool
}

func process(param LifecycleParameter) error {
	static := param.Config.Spa.Static
	logger := param.Logger
	preprocessors := param.Preprocessors
	pool := param.Pool
	return filepath.Walk(static, func(path string, info fs.FileInfo, err error) error {
		logger.Debugf("walk at %s", path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		mtype := pkg.DetectMIME(path)
		logger.Debugf("mimetype %s", mtype)
		for _, p := range preprocessors {
			if p.CanProcess(path, mtype) {
				logger.Debugf("find preprocessor %s", p.Name())
				err := pool.Submit(func() {
					err := p.Process(path)
					if err != nil {
						logger.Warnf("preprocessor error %e", err)
					}
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
