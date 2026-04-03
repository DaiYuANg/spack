package config

import (
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
				dix.TypedService[*Http](),
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
	dix.ProvideValueT(c, &cfg.Http)
	dix.ProvideValueT(c, &cfg.Assets)
	dix.ProvideValueT(c, &cfg.Compression)

	return nil
}

func loadConfig() (*Config, error) {
	def := defaultConfig()
	loaded, err := configx.LoadTErr[Config](
		configx.WithTypedDefaults(def),
		configx.WithEnvPrefix(constant.EnvPrefix),
	)
	if err != nil {
		return nil, err
	}
	return &loaded, nil
}

func Load() (*Config, error) {
	return loadConfig()
}
