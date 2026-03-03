package config

type Compression struct {
	Enable        bool   `koanf:"enable"`
	CacheDir      string `koanf:"cache_dir"`
	MinSize       int64  `koanf:"min_size"`
	Workers       int    `koanf:"workers"`
	BrotliQuality int    `koanf:"brotli_quality"`
	GzipLevel     int    `koanf:"gzip_level"`
}
