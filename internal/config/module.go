package config

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/validation"
	"github.com/go-playground/validator/v10"
	"github.com/samber/oops"
)

func NewModule(loadOptions LoadOptions) dix.Module {
	return dix.NewModule("config",
		dix.WithModuleProviders(
			dix.ProviderErr1(func(validate *validator.Validate) (*Config, error) {
				return loadConfig(loadOptions, validate)
			}),
			dix.Provider1(func(cfg *Config) *Debug { return &cfg.Debug }),
			dix.Provider1(func(cfg *Config) *Image { return &cfg.Image }),
			dix.Provider1(func(cfg *Config) *Metrics { return &cfg.Metrics }),
			dix.Provider1(func(cfg *Config) *Logger { return &cfg.Logger }),
			dix.Provider1(func(cfg *Config) *Robots { return &cfg.Robots }),
			dix.Provider1(func(cfg *Config) *HTTP { return &cfg.HTTP }),
			dix.Provider1(func(cfg *Config) *Assets { return &cfg.Assets }),
			dix.Provider1(func(cfg *Config) *Async { return &cfg.Async }),
			dix.Provider1(func(cfg *Config) *Compression { return &cfg.Compression }),
		),
	)
}

func loadConfig(loadOptions LoadOptions, validate *validator.Validate) (*Config, error) {
	loaded := defaultConfig()
	err := configx.Load(&loaded, loadOptions.configxOptions(validate)...)
	if err != nil {
		return nil, oops.In("config").Wrap(fmt.Errorf("load config: %w", err))
	}
	return &loaded, nil
}

func loadConfigWithValidation(loadOptions LoadOptions) (*Config, error) {
	validate, err := validation.New()
	if err != nil {
		return nil, oops.In("config").Wrap(fmt.Errorf("build validator: %w", err))
	}
	return loadConfig(loadOptions, validate)
}

func LoadWithOptions(loadOptions LoadOptions) (*Config, error) {
	return loadConfigWithValidation(loadOptions)
}
