package lifecycle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/preprocessor"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Parameter struct {
	fx.In
	Config        *config.Config
	Logger        *zap.SugaredLogger
	Preprocessors []preprocessor.Preprocessor `group:"preprocessor"`
	Lifecycle     fx.Lifecycle
	Pool          *ants.Pool
	Registry      registry.Registry
}

func startup(param Parameter) error {
	preprocessorConfig := param.Config.Preprocessor

	static := param.Config.Spa.Static
	logger := param.Logger
	preprocessors := param.Preprocessors
	pool := param.Pool
	r := param.Registry
	sort.SliceStable(preprocessors, func(i, j int) bool {
		return preprocessors[i].Order() < preprocessors[j].Order()
	})

	logger.Debugf("Preprocessors order: %v", lo.Map(preprocessors, func(p preprocessor.Preprocessor, _ int) string {
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

		originalInfo, err := generateOriginalInfo(path)
		if err != nil {
			return err
		}
		err = r.RegisterOriginal(originalInfo)
		if err != nil {
			return err
		}
		if !preprocessorConfig.Enable {
			return nil
		}
		// ✅ 每个文件作为一个任务提交到协程池
		err = pool.Submit(func() {
			for _, p := range preprocessors {
				if !p.CanProcess(originalInfo) {
					continue
				}
				logger.Debugf("run preprocessor %s on %s", p.Name(), path)
				if err := p.Process(originalInfo); err != nil {
					logger.Warnf("preprocessor %s error: %v", p.Name(), err)
					continue
				}
			}
		})
		if err != nil {
			return err
		}

		return nil
	})
}

func generateOriginalInfo(path string) (*registry.OriginalFileInfo, error) {
	mtype := pkg.DetectMIME(path)
	hash, err := pkg.FileHashWithExt(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(path)
	originalFileInfo := &registry.OriginalFileInfo{
		Path:     path,
		Size:     info.Size(),
		Hash:     hash,
		Ext:      ext,
		Mimetype: mtype,
	}

	return originalFileInfo, nil
}
