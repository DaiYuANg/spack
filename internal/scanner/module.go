package scanner

import (
	"context"
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"runtime"
	"sproxy/internal/config"
)

var Module = fx.Module("scanner",
	fx.Provide(
		asProcessor(newWebPConverter),
		asProcessor(newCompressor),
		newScanner,
	),
	fx.Invoke(runScan),
)

type Dependency struct {
	fx.In
	DB         *bbolt.DB
	Config     *config.Config
	Logger     *zap.SugaredLogger
	Processors []FileProcessor `group:"processor"`
}

func newScanner(dep Dependency) *Scanner {
	log := dep.Logger
	log.Debugf("processors: %v", len(dep.Processors))
	return &Scanner{
		root:       dep.Config.Spa.Static,
		processors: dep.Processors,
		maxWorkers: runtime.NumCPU() * 2,
	}
}

func runScan(scanner *Scanner) error {
	return scanner.Run(context.Background())
}
