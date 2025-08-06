package registry

import (
	"os"
	"path/filepath"

	"github.com/lucmq/go-shelve/shelve"
	"go.uber.org/zap"
)

type localDiskRegistry struct {
	data *shelve.Shelf[string, string]
}

func newLocalDiskRegistry(logger *zap.SugaredLogger) (*localDiskRegistry, error) {
	path := filepath.Join(os.TempDir(), "go-shelve")
	shelf, err := shelve.Open[string, string](path)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return &localDiskRegistry{
		data: shelf,
	}, nil
}
