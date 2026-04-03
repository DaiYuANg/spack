package logger

import (
	"context"
	"log/slog"

	dixadvanced "github.com/DaiYuANg/arcgo/dix/advanced"
	"github.com/DaiYuANg/arcgo/logx"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("logger",
	dix.WithModuleSetups(
		dixadvanced.Override1(buildLogger),
	),
	dix.WithModuleHooks(
		dix.OnStop(func(ctx context.Context, logger *slog.Logger) error {
			return logx.Close(logger)
		}),
	),
)

func buildLogger(cfg *config.Config) *slog.Logger {
	opts := []logx.Option{
		logx.WithLevelString(cfg.Logger.Level),
		logx.WithConsole(cfg.Logger.Console.Enabled),
		logx.WithCaller(true),
		logx.WithGlobalLogger(),
	}

	if cfg.Logger.File.Enabled {
		opts = append(opts,
			logx.WithFile(cfg.Logger.File.Path),
			logx.WithFileRotation(cfg.Logger.File.MaxSize, cfg.Logger.File.MaxAge, cfg.Logger.File.MaxFiles),
		)
	}

	logger, err := logx.New(opts...)
	if err != nil {
		fallback := slog.Default()
		fallback.Error("logger bootstrap failed, fallback to slog default", slog.String("err", err.Error()))
		return fallback
	}

	logx.SetDefault(logger)
	return logger
}
