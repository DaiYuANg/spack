package config

import (
	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/constant"
)

var Module = dix.NewModule("config",
	dix.WithModuleSetups(
		dix.Setup(setupConfig),
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
