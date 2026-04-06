package config

type Debug struct {
	Enable      bool   `koanf:"enable"`
	PprofPrefix string `koanf:"pprof_prefix" validate:"required,startswith=/"`
	LivePort    int    `koanf:"live_port"    validate:"gte=1,lte=65535"`
	Address     string `koanf:"address"`
}
