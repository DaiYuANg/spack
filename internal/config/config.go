package config

type Config struct {
	HTTP        HTTP        `koanf:"http"        validate:"required"`
	Assets      Assets      `koanf:"assets"      validate:"required"`
	Async       Async       `koanf:"async"       validate:"required"`
	Debug       Debug       `koanf:"debug"       validate:"required"`
	Image       Image       `koanf:"image"       validate:"required"`
	Metrics     Metrics     `koanf:"metrics"     validate:"required"`
	Logger      Logger      `koanf:"logger"      validate:"required"`
	Compression Compression `koanf:"compression" validate:"required"`
}

type Metrics struct {
	Prefix string `koanf:"prefix" validate:"required,startswith=/"`
}
