package config

import (
	"github.com/samber/lo"
	"strconv"
)

type Config struct {
	Http  Http  `koanf:"http"`
	Spa   Spa   `koanf:"spa"`
	Proxy Proxy `koanf:"proxy"`
	Debug Debug `koanf:"debug"`
	Limit Limit `koanf:"limit"`
}

type Http struct {
	Port int `koanf:"port"`
}

func (h Http) GetPort() string {
	return strconv.Itoa(h.Port)
}

type Spa struct {
	//Serve static spa config
	Static string `koanf:"static"`
	//default load file config like nginx try file
	Fallback    string      `koanf:"fallback"`
	Compression Compression `koanf:"compression"`
}

type Compression struct {
	Enabled    bool     `koanf:"enabled"`
	Algorithms []string `koanf:"algorithms"`
}

type Proxy struct {
	Path   string `koanf:"path"`
	Target string `koanf:"target"`
}

type Debug struct {
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
			Port: 80,
		},
		Spa: Spa{
			Fallback: "index.html",
			Compression: Compression{
				Enabled:    true,
				Algorithms: []string{"gzip"},
			},
		}}
}
