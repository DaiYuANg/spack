package cmd

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/eventbus"
	"github.com/daiyuang/spack/internal/lifecycle"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/pool"
	"github.com/daiyuang/spack/internal/processor"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func createContainer(userModules ...fx.Option) *fx.App {
	commonModules := []fx.Option{
		pool.Module,
		config.Module,
		logger.Module,
		processor.Module,
		lifecycle.Module,
		eventbus.Module,
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
	}

	allModules := append(commonModules, userModules...)

	return fx.New(allModules...)
}
