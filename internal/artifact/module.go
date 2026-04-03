package artifact

import (
	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module("artifact", fx.Provide(
	func(cfg *config.Compression) Store {
		return newLocalStore(cfg.CacheDir)
	},
))
