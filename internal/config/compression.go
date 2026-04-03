package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

const (
	CompressionModeOff    = "off"
	CompressionModeLazy   = "lazy"
	CompressionModeWarmup = "warmup"
)

type Compression struct {
	Enable         bool   `koanf:"enable"`
	Mode           string `koanf:"mode"`
	CacheDir       string `koanf:"cache_dir"`
	MinSize        int64  `koanf:"min_size"`
	Workers        int    `koanf:"workers"`
	QueueSize      int    `koanf:"queue_size"`
	CleanupEvery   string `koanf:"cleanup_every"`
	MaxAge         string `koanf:"max_age"`
	ImageMaxAge    string `koanf:"image_max_age"`
	EncodingMaxAge string `koanf:"encoding_max_age"`
	MaxCacheBytes  int64  `koanf:"max_cache_bytes"`
	BrotliQuality  int    `koanf:"brotli_quality"`
	GzipLevel      int    `koanf:"gzip_level"`
}

func (c Compression) NormalizedMode() string {
	switch strings.ToLower(strings.TrimSpace(c.Mode)) {
	case "", CompressionModeLazy:
		return CompressionModeLazy
	case CompressionModeOff:
		return CompressionModeOff
	case CompressionModeWarmup:
		return CompressionModeWarmup
	default:
		return CompressionModeLazy
	}
}

func (c Compression) PipelineEnabled() bool {
	return c.Enable && c.NormalizedMode() != CompressionModeOff
}

func (c Compression) QueueCapacity() int {
	if c.QueueSize > 0 {
		return c.QueueSize
	}
	workers := c.Workers
	if workers < 1 {
		workers = 1
	}
	return workers * 64
}

func (c Compression) ParsedCleanupInterval() time.Duration {
	raw := strings.TrimSpace(c.CleanupEvery)
	if raw == "" {
		return 5 * time.Minute
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 5 * time.Minute
	}
	return d
}

func (c Compression) ParsedMaxAge() time.Duration {
	return parseFlexibleDuration(c.MaxAge)
}

func (c Compression) NamespaceMaxAges() collectionx.Map[string, time.Duration] {
	out := collectionx.NewMapWithCapacity[string, time.Duration](2)
	if d := parseFlexibleDuration(c.EncodingMaxAge); d > 0 {
		out.Set("encoding", d)
	}
	if d := parseFlexibleDuration(c.ImageMaxAge); d > 0 {
		out.Set("image", d)
	}
	return out
}

func parseFlexibleDuration(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err == nil {
		if d > 0 {
			return d
		}
		return 0
	}

	seconds, secErr := strconv.ParseInt(raw, 10, 64)
	if secErr != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
