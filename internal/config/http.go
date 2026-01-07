package config

import "strconv"

type Http struct {
	Port      int  `koanf:"port"`
	LowMemory bool `koanf:"low_memory"`
}

func (h Http) GetPort() string {
	return strconv.Itoa(h.Port)
}
