package config

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/validation"
	"github.com/go-playground/validator/v10"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
)

var Module = NewModule(LoadOptions{})

func NewModule(loadOptions LoadOptions) dix.Module {
	return dix.NewModule("config",
		dix.WithModuleProviders(
			dix.RawProviderWithMetadata(registerConfigProvider(loadOptions), dix.ProviderMetadata{
				Label:  "ConfigProvider",
				Output: dix.TypedService[*Config](),
				Dependencies: dix.ServiceRefs(
					dix.TypedService[*validator.Validate](),
				),
				Raw: true,
			}),
			dix.Provider1(func(cfg *Config) *Debug { return &cfg.Debug }),
			dix.Provider1(func(cfg *Config) *Image { return &cfg.Image }),
			dix.Provider1(func(cfg *Config) *Metrics { return &cfg.Metrics }),
			dix.Provider1(func(cfg *Config) *Logger { return &cfg.Logger }),
			dix.Provider1(func(cfg *Config) *HTTP { return &cfg.HTTP }),
			dix.Provider1(func(cfg *Config) *Assets { return &cfg.Assets }),
			dix.Provider1(func(cfg *Config) *Async { return &cfg.Async }),
			dix.Provider1(func(cfg *Config) *Compression { return &cfg.Compression }),
		),
	)
}

func registerConfigProvider(loadOptions LoadOptions) func(*dix.Container) {
	return func(c *dix.Container) {
		do.ProvideNamed(c.Raw(), dix.TypedService[*Config]().Name, func(i do.Injector) (*Config, error) {
			validate, err := do.InvokeNamed[*validator.Validate](i, dix.TypedService[*validator.Validate]().Name)
			if err != nil {
				return nil, err
			}
			return loadConfig(loadOptions, validate)
		})
	}
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

func Load() (*Config, error) {
	return loadConfigWithValidation(LoadOptions{})
}

func LoadWithOptions(loadOptions LoadOptions) (*Config, error) {
	return loadConfigWithValidation(loadOptions)
}
