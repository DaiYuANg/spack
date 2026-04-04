package source

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
)

// NewLocalFSForTest exposes local filesystem source construction for external tests.
func NewLocalFSForTest(cfg *config.Assets, logger *slog.Logger) (Source, error) {
	return newLocalFS(cfg, logger)
}
