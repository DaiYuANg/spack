package printer

import (
	"context"
	"log/slog"
	"time"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/registry"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"printer",
	fx.Invoke(run),
)

func run(
	r registry.Registry,
	cfg *config.Config,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) {
	if !cfg.Printer.Enable {
		logger.Debug("printer disabled by config")
		return
	}

	// 通过上下文取消控制轮询协程
	ctx, cancel := context.WithCancel(context.Background())
	lifecycle.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cancel() // 应用退出时取消轮询
			return nil
		},
	})

	go waitForFreezeAndPrint(ctx, r, cfg, logger)
}

func waitForFreezeAndPrint(
	ctx context.Context,
	r registry.Registry,
	cfg *config.Config,
	logger *slog.Logger,
) {
	const pollInterval = 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return // 应用退出 -> 停止轮询
		default:
			// 检查 frozen
			if r.IsFrozen() {
				logger.Info("registry frozen — running printer")

				// 执行打印并退出循环
				switch cfg.Printer.Mode {
				case config.ModeJSON:
					PrintJSON(r, logger)
				default:
					PrintTable(r, logger)
				}
				return
			}
			time.Sleep(pollInterval)
		}
	}
}
