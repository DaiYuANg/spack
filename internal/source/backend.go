package source

import (
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/config"
)

type sourceFactory func(*config.Assets, *slog.Logger) (Source, error)

var sourceFactories = collectionx.NewMapFrom[config.SourceBackend, sourceFactory](map[config.SourceBackend]sourceFactory{
	config.SourceBackendLocal: newLocalFS,
})

func newSourceFromConfig(cfg *config.Assets, logger *slog.Logger) (Source, error) {
	backend := cfg.NormalizedBackend()
	factory, ok := sourceFactories.Get(backend)
	if !ok || !config.IsSupportedSourceBackend(backend) {
		return nil, fmt.Errorf(
			"unsupported assets backend: %s (supported: %s)",
			backend,
			config.SupportedSourceBackendNames().Join(", "),
		)
	}
	return factory(cfg, logger)
}
