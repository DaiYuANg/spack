package cmd

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/pool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createContainer(userModules ...fx.Option) *fx.App {
	// 公共模块：主命令和子命令都需要的部分
	commonModules := []fx.Option{
		pool.Module,
		config.Module,
		logger.Module,
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			fxLogger := &fxevent.ZapLogger{Logger: log}
			fxLogger.UseLogLevel(zapcore.DebugLevel)
			return fxLogger
		}),
	}

	// 将公共模块和用户传入的模块组合起来
	allModules := append(commonModules, userModules...)

	return fx.New(allModules...)
}
