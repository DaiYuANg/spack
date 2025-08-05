package cmd

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/http"
	"github.com/daiyuang/spack/internal/logger"
	"github.com/daiyuang/spack/internal/metrics"
	"github.com/daiyuang/spack/internal/preprocessor"
	"github.com/daiyuang/spack/internal/prometheus"
	"github.com/daiyuang/spack/internal/registry"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func container() *fx.App {
	return fx.New(
		config.Module,
		logger.Module,
		registry.Module,
		metrics.Module,
		prometheus.Module,
		http.Module,
		preprocessor.Module,
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			fxLogger := &fxevent.ZapLogger{Logger: log}
			fxLogger.UseLogLevel(zapcore.DebugLevel)
			return fxLogger
		}),
	)
}
