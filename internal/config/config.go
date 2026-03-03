package config

type Config struct {
	Http        Http        `koanf:"http"`
	Cache       Cache       `koanf:"cache"`
	Assets      Assets      `koanf:"assets"`
	Debug       Debug       `koanf:"debug"`
	Limit       Limit       `koanf:"limit"`
	Metrics     Metrics     `koanf:"metrics"`
	Logger      Logger      `koanf:"logger"`
	Processor   Processor   `koanf:"scanner"`
	Compression Compression `koanf:"compression"`
}

type Metrics struct {
	Prefix string `koanf:"prefix"`
}

type Limit struct {
	Enable bool `koanf:"enable"`
}

type Processor struct {
	Enable bool `koanf:"enable"`
}
