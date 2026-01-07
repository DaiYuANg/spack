package config

import (
	"github.com/samber/lo"
)

type Config struct {
	Http       Http       `koanf:"http"`
	Cache      Cache      `koanf:"cache"`
	Spa        Spa        `koanf:"spa"`
	Proxy      Proxy      `koanf:"proxy"`
	Debug      Debug      `koanf:"debug"`
	Limit      Limit      `koanf:"limit"`
	Prometheus Prometheus `koanf:"prometheus"`
	Logger     Logger     `koanf:"logger"`
	Processor  Processor  `koanf:"scanner"`
}

type Prometheus struct {
	Prefix string `koanf:"prefix"`
}

type Proxy struct {
	Path   string `koanf:"path"`
	Target string `koanf:"target"`
}

type Limit struct {
	Enable bool `koanf:"enable"`
}

type Processor struct {
	Enable bool `koanf:"enable"`
}

func (p Proxy) Enabled() bool {
	return lo.IsNotEmpty(p.Path) && lo.IsNotEmpty(p.Target)
}
