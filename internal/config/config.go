package config

import (
	"github.com/samber/lo"
	"strconv"
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
}

type Logger struct {
	Level string `koanf:"level"`
}

type Cache struct {
	Max int64 `koanf:"max"`
}

type Http struct {
	Port      int  `koanf:"port"`
	Prefork   bool `koanf:"prefork"`
	LowMemory bool `koanf:"low_memory"`
}

func (h Http) GetPort() string {
	return strconv.Itoa(h.Port)
}

type Spa struct {
	//Serve static spa config
	Static string `koanf:"static"`
	//default load file config like nginx try file
	Fallback string `koanf:"fallback"`
}

type Prometheus struct {
	Prefix string `koanf:"prefix"`
}

type Proxy struct {
	Path   string `koanf:"path"`
	Target string `koanf:"target"`
}

type Debug struct {
	Prefix string `koanf:"prefix"`
}

type Monitor struct {
	Prefix string `koanf:"prefix"`
}

type Limit struct {
	Enable bool `koanf:"enable"`
}

func (p Proxy) Enabled() bool {
	return lo.IsNotEmpty(p.Path) && lo.IsNotEmpty(p.Target)
}

func defaultConfig() Config {
	return Config{
		Http: Http{
			Port:      80,
			Prefork:   false,
			LowMemory: true,
		},
		Spa: Spa{
			Fallback: "index.html",
		},
	}
}
