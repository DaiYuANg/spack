package scanner

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module("scanner",
	fx.Provide(
		newLocalFsBackendInstance,
		NewScanner,
	),
)

func newLocalFsBackendInstance(config *config.Config, logger *slog.Logger) Backend {
	return NewLocalFSBackend(config.Spa.Static, logger)
}
