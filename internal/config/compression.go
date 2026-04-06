package config

import (
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/validation"
)

const (
	CompressionModeOff    = "off"
	CompressionModeLazy   = "lazy"
	CompressionModeWarmup = "warmup"
)

type Compression struct {
	Enable         bool   `koanf:"enable"`
	Mode           string `koanf:"mode"             validate:"required,oneof=off lazy warmup"`
	CacheDir       string `koanf:"cache_dir"        validate:"required"`
	MinSize        int64  `koanf:"min_size"         validate:"gte=0"`
	Workers        int    `koanf:"workers"          validate:"gte=0"`
	QueueSize      int    `koanf:"queue_size"       validate:"gte=0"`
	CleanupEvery   string `koanf:"cleanup_every"    validate:"omitempty,spack_duration"`
	MaxAge         string `koanf:"max_age"          validate:"omitempty,spack_flexible_duration"`
	ImageMaxAge    string `koanf:"image_max_age"    validate:"omitempty,spack_flexible_duration"`
	EncodingMaxAge string `koanf:"encoding_max_age" validate:"omitempty,spack_flexible_duration"`
	MaxCacheBytes  int64  `koanf:"max_cache_bytes"  validate:"gte=0"`
	BrotliQuality  int    `koanf:"brotli_quality"   validate:"gte=0,lte=11"`
	GzipLevel      int    `koanf:"gzip_level"       validate:"gte=-2,lte=9"`
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
	workers := max(c.Workers, 1)
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
	return validation.ParseFlexibleDuration(c.MaxAge)
}

func (c Compression) NamespaceMaxAges() collectionx.Map[string, time.Duration] {
	out := collectionx.NewMapWithCapacity[string, time.Duration](2)
	if d := validation.ParseFlexibleDuration(c.EncodingMaxAge); d > 0 {
		out.Set("encoding", d)
	}
	if d := validation.ParseFlexibleDuration(c.ImageMaxAge); d > 0 {
		out.Set("image", d)
	}
	return out
}
