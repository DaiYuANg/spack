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

	const ioFactor = 20

	poolSize := cpuNum * ioFactor

	antsLogger := &Logger{
		logger: logger,
	}

	return ants.NewPool(
		poolSize,
		ants.WithExpiryDuration(10*time.Second), // 空闲worker 10s回收，减少资源占用
		ants.WithNonblocking(false),             // 任务提交满时阻塞
		ants.WithNonblocking(true),
		ants.WithLogger(antsLogger),
	)
}
