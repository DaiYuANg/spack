package cmd

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/lifecycle"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/pool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createContainer(userModules ...fx.Option) *fx.App {
	commonModules := []fx.Option{
		pool.Module,
		config.Module,
		logger.Module,
		lifecycle.Module,
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			fxLogger := &fxevent.ZapLogger{Logger: log}
			fxLogger.UseLogLevel(zapcore.DebugLevel)
			return fxLogger
		}),
	}

	allModules := append(commonModules, userModules...)

	return fx.New(allModules...)
}
