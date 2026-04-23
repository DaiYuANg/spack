// Package logger configures structured logging.
package logger

import (
	"context"
	"log/slog"

	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
	"github.com/arcgolabs/logx"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("logger",
	dix.WithModuleProviders(
		dix.Provider1(Build),
	),
	dix.WithModuleHooks(
		dix.OnStop(func(ctx context.Context, logger *slog.Logger) error {
			return logx.Close(logger)
		}),
	),
)

func Build(cfg *config.Config) *slog.Logger {
	opts := collectionx.NewListWithCapacity[logx.Option](5,
		logx.WithLevelString(cfg.Logger.Level),
		logx.WithConsole(cfg.Logger.Console.Enabled),
		logx.WithCaller(true),
		logx.WithGlobalLogger(),
	)

	if cfg.Logger.File.Enabled {
		opts.Add(
			logx.WithFile(cfg.Logger.File.Path),
			logx.WithFileRotation(cfg.Logger.File.MaxSize, cfg.Logger.File.MaxAge, cfg.Logger.File.MaxFiles),
		)
	}

	logger, err := logx.New(opts.Values()...)
	if err != nil {
		fallback := slog.Default()
		fallback.Error("logger bootstrap failed, fallback to slog default", slog.String("err", err.Error()))
		return fallback
	}

	logx.SetDefault(logger)
	return logger
}
