package cmd

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sproxy/internal/config"
	"sproxy/internal/http"
	"sproxy/internal/logger"
	"sproxy/internal/prometheus"
)

func container() *fx.App {
	return fx.New(
		config.Module,
		logger.Module,
		prometheus.Module,
		http.Module,
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			fxLogger := &fxevent.ZapLogger{Logger: log}
			fxLogger.UseLogLevel(zapcore.DebugLevel)
			return fxLogger
		}),
	)
}
