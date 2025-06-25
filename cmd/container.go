package cmd

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"sproxy/internal/config"
	"sproxy/internal/http"
	"sproxy/internal/logger"
)

func container() *fx.App {
	return fx.New(
		config.Module,
		logger.Module,
		http.Module,
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)
}
