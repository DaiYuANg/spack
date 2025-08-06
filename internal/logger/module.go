package logger

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Module = fx.Module("logger_module",
	fx.Provide(
		newLogger,
		sugaredLogger,
	),
	fx.Invoke(deferLogger),
)

func newLogger(cfg *config.Config) *zap.Logger {
	loggerConfig := cfg.Logger
	encoderCfg := zap.NewProductionEncoderConfig()

	// 时间格式化，精确到毫秒
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	var level zapcore.Level
	switch loggerConfig.Level {
	case "info":
		level = zapcore.InfoLevel
	case "debug":
		level = zapcore.DebugLevel
	default:
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		level,
	)

	logger := zap.New(
		core,
		zap.AddCaller(),                       // 显示文件名和行号
		zap.AddStacktrace(zapcore.ErrorLevel), // 错误及以上打印堆栈
	)

	return logger
}

func sugaredLogger(log *zap.Logger) *zap.SugaredLogger {
	return log.Sugar()
}

func deferLogger(lc fx.Lifecycle, logger *zap.Logger) {
	lc.Append(
		fx.StopHook(func() error {
			if err := logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
				return fmt.Errorf("logger sync failed: %v", err)
			}
			return nil
		}),
	)
}
