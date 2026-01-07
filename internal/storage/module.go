package storage

import (
	"log/slog"
	"os"

	"github.com/spf13/afero"
	"go.uber.org/fx"
)

var Module = fx.Module("storage",
	fx.Provide(
		newLocal,
	),
)

func newLocal(logger *slog.Logger) (*LocalFS, error) {
	return NewLocalFS(afero.NewOsFs(), os.TempDir(), logger)
}
