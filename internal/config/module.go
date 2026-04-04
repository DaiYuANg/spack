package config

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/samber/do/v2"
)

var Module = dix.NewModule("config",
	dix.WithModuleProviders(
		dix.RawProviderWithMetadata(registerConfigProvider, dix.ProviderMetadata{
			Label:  "ConfigProvider",
			Output: dix.TypedService[*Config](),
			Raw:    true,
		}),
		dix.Provider1(func(cfg *Config) *Debug { return &cfg.Debug }),
		dix.Provider1(func(cfg *Config) *Image { return &cfg.Image }),
		dix.Provider1(func(cfg *Config) *Metrics { return &cfg.Metrics }),
		dix.Provider1(func(cfg *Config) *Logger { return &cfg.Logger }),
		dix.Provider1(func(cfg *Config) *HTTP { return &cfg.HTTP }),
		dix.Provider1(func(cfg *Config) *Assets { return &cfg.Assets }),
		dix.Provider1(func(cfg *Config) *Compression { return &cfg.Compression }),
	),
)

func registerConfigProvider(c *dix.Container) {
	do.ProvideNamed(c.Raw(), dix.TypedService[*Config]().Name, func(do.Injector) (*Config, error) {
		return loadConfig()
	})
}

func loadConfig() (*Config, error) {
	loaded := defaultConfig()
	err := configx.Load(
		&loaded,
		configx.WithEnvPrefix(constant.EnvPrefix),
		configx.WithIgnoreDotenvError(true),
		configx.WithDotenv(),
	)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return &loaded, nil
}

func Load() (*Config, error) {
	return loadConfig()
}
