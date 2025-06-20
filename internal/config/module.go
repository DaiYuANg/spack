package config

import (
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"sproxy/internal/constant"
	"strings"
)

var Module = fx.Module("config", fx.Provide(newKoanf, loadConfig))

func newKoanf() *koanf.Koanf {
	return koanf.New(".")
}

func loadConfig(k *koanf.Koanf, logger *zap.SugaredLogger) (*Config, error) {
	def := defaultConfig()
	err := k.Load(structs.Provider(def, "koanf"), nil)
	if err != nil {
		return nil, err
	}
	err = k.Load(env.Provider(constant.EnvPrefix, ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, constant.EnvPrefix)), "_", ".", -1)
	}), nil)
	if err != nil {
		return nil, err
	}
	logger.Debugf("all key%s", k.All())
	err = k.Unmarshal("", &def)
	if err != nil {
		return nil, err
	}
	logger.Infof("loaded config %v", def)
	return &def, nil
}
