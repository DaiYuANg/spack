package preprocessor

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type LifecycleParameter struct {
	fx.In
	Config        *config.Config
	Logger        *zap.SugaredLogger
	Preprocessors []Preprocessor `group:"preprocessor"`
	Lifecycle     fx.Lifecycle
	Pool          *ants.Pool
}

func preprocess(param LifecycleParameter) error {
	preprocessorConfig := param.Config.Preprocessor
	if !preprocessorConfig.Enable {
		return nil
	}
	static := param.Config.Spa.Static
	logger := param.Logger
	preprocessors := param.Preprocessors
	pool := param.Pool
	// ✅ 按照 Order 排序（从小到大）
	sort.SliceStable(preprocessors, func(i, j int) bool {
		return preprocessors[i].Order() < preprocessors[j].Order()
	})

	logger.Debugf("Preprocessors order: %v", lo.Map(preprocessors, func(p Preprocessor, _ int) string {
		return fmt.Sprintf("%s(%d)", p.Name(), p.Order())
	}))

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
		// ✅ 每个文件作为一个任务提交到协程池
		err = pool.Submit(func() {
			for _, p := range preprocessors {
				if p.CanProcess(path, mtype) {
					logger.Debugf("run preprocessor %s on %s", p.Name(), path)
					if err := p.Process(path); err != nil {
						logger.Warnf("preprocessor %s error: %v", p.Name(), err)
						continue
					}
				}
			}
		})
		if err != nil {
			return err
		}

		return nil
	})
}
