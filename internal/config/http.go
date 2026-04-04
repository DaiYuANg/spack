package config

import (
	"strconv"
	"strings"
	"time"
)

type HTTP struct {
	Port        int         `koanf:"port"`
	LowMemory   bool        `koanf:"low_memory"`
	MemoryCache MemoryCache `koanf:"memory_cache"`
}

type MemoryCache struct {
	Enable      bool   `koanf:"enable"`
	Warmup      bool   `koanf:"warmup"`
	MaxEntries  int    `koanf:"max_entries"`
	MaxFileSize int64  `koanf:"max_file_size"`
	TTL         string `koanf:"ttl"`
}

func (h HTTP) GetPort() string {
	return strconv.Itoa(h.Port)
}

func (c MemoryCache) Enabled() bool {
	return c.Enable && c.MaxEntries > 0 && c.MaxFileSize > 0 && c.ParsedTTL() > 0
}

func (c MemoryCache) WarmupEnabled() bool {
	return c.Enabled() && c.Warmup
}

func (c MemoryCache) ParsedTTL() time.Duration {
	raw := strings.TrimSpace(c.TTL)
	if raw == "" {
		return 5 * time.Minute
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 5 * time.Minute
	}
	return d
}
