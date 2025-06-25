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

var Module = fx.Module("config", fx.Provide(
	newKoanf,
	loadConfig,
))

func newKoanf() *koanf.Koanf {
	return koanf.New(".")
}

func loadConfig(k *koanf.Koanf, logger *zap.SugaredLogger) (*Config, error) {
	def := defaultConfig()

	// 加载默认配置
	if err := k.Load(structs.Provider(def, "koanf"), nil); err != nil {
		return nil, err
	}

	// 使用 lo.Ternary 优化字符串映射函数
	mapEnvKey := func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, constant.EnvPrefix)), "_", ".")
	}
	if err := k.Load(env.Provider(constant.EnvPrefix, ".", mapEnvKey), nil); err != nil {
		return nil, err
	}

	allKeys := k.All()
	logger.Debugf("all key: %v", allKeys)

	if err := k.Unmarshal("", &def); err != nil {
		return nil, err
	}

	logger.Infof("loaded config: %+v", def)
	return &def, nil
}
