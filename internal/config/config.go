package config

type Config struct {
	Http        Http        `koanf:"http"`
	Assets      Assets      `koanf:"assets"`
	Debug       Debug       `koanf:"debug"`
	Image       Image       `koanf:"image"`
	Metrics     Metrics     `koanf:"metrics"`
	Logger      Logger      `koanf:"logger"`
	Compression Compression `koanf:"compression"`
}

type Metrics struct {
	Prefix string `koanf:"prefix"`
}
