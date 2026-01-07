package config

type Cache struct {
	Max  int64  `koanf:"max"`
	Type string `koanf:"type"`
}
