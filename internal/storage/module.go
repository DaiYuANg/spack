package storage

import (
	"log/slog"
	"os"
	"path"

	"github.com/spf13/afero"
	"go.uber.org/fx"
)

var Module = fx.Module("storage",
	fx.Provide(
		newLocal,
	),
)

func newLocal(logger *slog.Logger) (*LocalFS, error) {
	fullpath := path.Join(os.TempDir(), "spack")
	return NewLocalFS(afero.NewOsFs(), fullpath, logger)
}
