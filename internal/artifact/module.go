// Package artifact manages generated artifact storage.
package artifact

import (
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("artifact",
	dix.WithModuleProviders(
		dix.Provider1(func(cfg *config.Compression) Store {
			return newLocalStore(cfg.CacheDir)
		}),
	),
)
