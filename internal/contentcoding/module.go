package contentcoding

import (
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("contentcoding",
	dix.WithModuleProviders(
		dix.Provider1(func(cfg *config.Compression) Registry {
			return NewRegistry(Options{
				BrotliQuality: cfg.BrotliQuality,
				GzipLevel:     cfg.GzipLevel,
				ZstdLevel:     cfg.ZstdLevel,
			}, cfg.NormalizedEncodings())
		}),
	),
)
