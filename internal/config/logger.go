package config

type Console struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

type File struct {
	Enabled  bool   `json:"enabled"   validate:"-"                        yaml:"enabled"`
	Path     string `json:"path"      validate:"required_if=Enabled true" yaml:"path"`
	MaxSize  int    `json:"max_size"  validate:"gte=0"                    yaml:"max_size"`
	MaxAge   int    `json:"max_age"   validate:"gte=0"                    yaml:"max_age"`
	MaxFiles int    `json:"max_files" validate:"gte=0"                    yaml:"max_files"`
}

type Logger struct {
	Level   string  `json:"level"   validate:"required,oneof=debug info warn error" yaml:"level"`
	Console Console `json:"console" yaml:"console"`
	File    File    `json:"file"    yaml:"file"`
}
