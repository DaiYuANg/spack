package config

import "strconv"

type HTTP struct {
	Port      int  `koanf:"port"`
	LowMemory bool `koanf:"low_memory"`
}

func (h HTTP) GetPort() string {
	return strconv.Itoa(h.Port)
}
