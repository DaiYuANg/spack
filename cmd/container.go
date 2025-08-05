package cmd

import (
	"github.com/daiyuang/spack/internal/cache"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/http"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/metrics"
	"github.com/daiyuang/spack/internal/preprocessor"
	"github.com/daiyuang/spack/internal/printer"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newAnts() (*ants.Pool, error) {
	return ants.NewPool(10000)
}
func container() *fx.App {
	return fx.New(
		fx.Provide(newAnts),
		config.Module,
		cache.Module,
		logger.Module,
		registry.Module,
		preprocessor.Module,
		http.Module,
		metrics.Module,
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			fxLogger := &fxevent.ZapLogger{Logger: log}
			fxLogger.UseLogLevel(zapcore.DebugLevel)
			return fxLogger
		}),
		printer.Module,
	)
}

func createContainer(userModules ...fx.Option) *fx.App {
	// 公共模块：主命令和子命令都需要的部分
	commonModules := []fx.Option{
		fx.Provide(newAnts),
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
