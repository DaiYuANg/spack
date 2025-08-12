package pool

import (
	"runtime"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("pool", fx.Provide(
	newAnts,
))

type Logger struct {
	logger *zap.SugaredLogger
}

func (l *Logger) Printf(format string, args ...any) {
	l.logger.Debugf(format, args...)
}

func newAnts(logger *zap.SugaredLogger) (*ants.Pool, error) {
	cpuNum := runtime.NumCPU()

	const ioFactor = 200

	poolSize := cpuNum * ioFactor

	antsLogger := &Logger{
		logger: logger,
	}

	return ants.NewPool(
		poolSize,
		ants.WithExpiryDuration(10*time.Second),
		ants.WithNonblocking(false),
		ants.WithLogger(antsLogger),
	)
}
