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
	Static   string `koanf:"static"`
	Fallback string `koanf:"fallback"`
	Image    Image  `koanf:"image"`
}

type Image struct {
	Webp bool `koanf:"webp"`
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
		}}
}
