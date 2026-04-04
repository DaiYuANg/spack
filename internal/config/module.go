package config

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/constant"
)

var Module = dix.NewModule("config",
	dix.WithModuleSetups(
		dix.SetupWithMetadata(setupConfig, dix.SetupMetadata{
			Label: "SetupConfig",
			Provides: dix.ServiceRefs(
				dix.TypedService[*Config](),
				dix.TypedService[*Debug](),
				dix.TypedService[*Image](),
				dix.TypedService[*Metrics](),
				dix.TypedService[*Logger](),
				dix.TypedService[*HTTP](),
				dix.TypedService[*Assets](),
				dix.TypedService[*Compression](),
			),
			GraphMutation: true,
		}),
	),
)

func setupConfig(c *dix.Container, _ dix.Lifecycle) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	dix.ProvideValueT(c, cfg)
	dix.ProvideValueT(c, &cfg.Debug)
	dix.ProvideValueT(c, &cfg.Image)
	dix.ProvideValueT(c, &cfg.Metrics)
	dix.ProvideValueT(c, &cfg.Logger)
	dix.ProvideValueT(c, &cfg.HTTP)
	dix.ProvideValueT(c, &cfg.Assets)
	dix.ProvideValueT(c, &cfg.Compression)

	return nil
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
