package lifecycle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/daiyuang/spack/pkg"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Parameter struct {
	fx.In
	Config        *config.Config
	Logger        *zap.SugaredLogger
	Preprocessors []scanner.Preprocessor `group:"scanner"`
	Lifecycle     fx.Lifecycle
	Pool          *ants.Pool
	Registry      registry.Registry
}

func startup(param Parameter) {
	cfg := param.Config
	logger := param.Logger
	err := func() error {
		preprocessors := sortPreprocessors(param.Preprocessors)
		logPreprocessorOrder(logger, preprocessors)

		return filepath.Walk(cfg.Spa.Static, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				e := oops.In("Preprocessor lifecycle").Wrap(err)
				return e
			}
			if info.IsDir() {
				return nil
			}
			return handleFile(path, cfg.Preprocessor, preprocessors, param)
		})
	}()
	if err != nil {
		logger.Warnf("startup error: %v", err)
	}

}

// 排序预处理器
func sortPreprocessors(pp []scanner.Preprocessor) []scanner.Preprocessor {
	sort.SliceStable(pp, func(i, j int) bool {
		return pp[i].Order() < pp[j].Order()
	})
	return pp
}

// 打印预处理器顺序
func logPreprocessorOrder(logger *zap.SugaredLogger, pp []scanner.Preprocessor) {
	logger.Debugf("Preprocessors order: %v", lo.Map(pp, func(p scanner.Preprocessor, _ int) string {
		return fmt.Sprintf("%s(%d)", p.Name(), p.Order())
	}))
}

// 处理单个文件
func handleFile(path string, cfg config.Preprocessor, pp []scanner.Preprocessor, param Parameter) error {
	logger := param.Logger
	r := param.Registry
	pool := param.Pool

	originalInfo, err := generateOriginalInfo(path)
	if err != nil {
		return err
	}

	if err := r.Writer().RegisterOriginal(originalInfo); err != nil {
		return err
	}

	if !cfg.Enable {
		return nil
	}

	return pool.Submit(func() {
		runPreprocessors(pp, originalInfo, path, logger)
	})
}

// 执行可处理的预处理器
func runPreprocessors(pp []scanner.Preprocessor, info *registry.OriginalFileInfo, path string, logger *zap.SugaredLogger) {
	lo.ForEach(
		lo.Filter(pp, func(p scanner.Preprocessor, _ int) bool {
			return p.CanProcess(info)
		}),
		func(p scanner.Preprocessor, _ int) {
			logger.Debugf("run scanner %s on %s", p.Name(), path)
			if err := p.Process(info); err != nil {
				logger.Warnf("scanner %s error: %v", p.Name(), err)
			}
		},
	)
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
