package lifecycle

import (
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/daiyuang/spack/internal/processor"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type ScanParameter struct {
	fx.In
	Scanner  *scanner.Scanner
	Registry registry.Registry
	Pps      []processor.Processor `group:"processor"`
	Pool     *ants.Pool
	Logger   *slog.Logger
}

func scan(parameter ScanParameter) error {
	scannerInstance := parameter.Scanner
	reg := parameter.Registry
	pool := parameter.Pool
	logger := parameter.Logger
	lo.ForEach(parameter.Pps, func(p processor.Processor, _ int) {
		logger.Info("Scanners", slog.String("scanner name", p.Name()))
	})
	var wg sync.WaitGroup
	var submitErr atomic.Pointer[error] // 可记录第一个 submit 错误
	err := scannerInstance.Scan(func(obj *scanner.ObjectInfo, hash string) error {
		// 1️⃣ 先注册原始文件
		info := &registry.OriginalFileInfo{
			Path:     obj.Key,
			Size:     obj.Size,
			Hash:     hash,
			Ext:      filepath.Ext(obj.Key),
			Mimetype: obj.Mimetype,
		}

		if err := reg.Writer().RegisterOriginal(info); err != nil {
			return oops.Wrap(err)
		}

		ctx := processor.Context{
			Obj:      obj,
			Hash:     hash,
			Registry: reg.Writer(),
			Open:     obj.Reader,
			EmitVariant: func(v *registry.VariantFileInfo) error {
				return reg.Writer().AddVariant(obj.Key, v)
			},
		}

		// 为每个 processor 生成任务
		lo.ForEach(parameter.Pps, func(p processor.Processor, _ int) {
			if !p.Match(obj) {
				return
			}

			wg.Add(1)
			submitErrVal := pool.Submit(func() {
				defer wg.Done()
				_, err := p.Run(ctx)
				if err != nil {
					logger.Error("processor error", oops.Wrap(err))
				}
			})
			if submitErrVal != nil {
				logger.Error("failed to submit task", oops.Wrap(submitErrVal))
				submitErr.Store(&submitErrVal)
				wg.Done() // 提交失败要减少计数
			}
		})

		return nil
	})
	if err != nil {
		return oops.Wrap(err)
	}
	wg.Wait()

	//registry freeze
	err = reg.Freeze()
	if err != nil {
		return oops.Wrap(err)
	}
	return nil
}
