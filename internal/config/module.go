package config

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/samber/do/v2"
)

var Module = NewModule(LoadOptions{})

func NewModule(loadOptions LoadOptions) dix.Module {
	return dix.NewModule("config",
		dix.WithModuleProviders(
			dix.RawProviderWithMetadata(registerConfigProvider(loadOptions), dix.ProviderMetadata{
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
}

func registerConfigProvider(loadOptions LoadOptions) func(*dix.Container) {
	return func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), dix.TypedService[*Config]().Name, func(do.Injector) (*Config, error) {
			return loadConfig(loadOptions)
		})
	}
}

func loadConfig(loadOptions LoadOptions) (*Config, error) {
	loaded := defaultConfig()
	err := configx.Load(&loaded, loadOptions.configxOptions()...)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return &loaded, nil
}

func Load() (*Config, error) {
	return loadConfig(LoadOptions{})
}

func LoadWithOptions(loadOptions LoadOptions) (*Config, error) {
	return loadConfig(loadOptions)
}
