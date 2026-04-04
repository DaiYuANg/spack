package config

import (
	"os"
	"path/filepath"
)

func defaultConfig() Config {
	return Config{
		HTTP: HTTP{
			Port:      80,
			LowMemory: true,
			MemoryCache: MemoryCache{
				Enable:      false,
				Warmup:      false,
				MaxEntries:  1024,
				MaxFileSize: 64 * 1024,
				TTL:         "5m",
			},
		},
		Assets: Assets{
			Path:     "/",
			Entry:    "index.html",
			Fallback: Fallback{On: FallbackOnNotFound, Target: "index.html"},
		},
		Logger: Logger{
			Level: "debug",
			Console: Console{
				Enabled: true,
			},
			File: File{Enabled: false},
		},
		Metrics: Metrics{Prefix: "/prometheus"},
		Debug: Debug{
			Enable:      true,
			PprofPrefix: "/pprof",
			LivePort:    8080,
		},
		Image: Image{
			Enable:      true,
			Widths:      "640,1280,1920",
			JPEGQuality: 78,
		},
		Compression: Compression{
			Mode:           CompressionModeLazy,
			Enable:         true,
			CacheDir:       filepath.Join(os.TempDir(), "spack-cache"),
			MinSize:        1024,
			Workers:        2,
			QueueSize:      128,
			CleanupEvery:   "5m",
			MaxAge:         "168h",
			ImageMaxAge:    "336h",
			EncodingMaxAge: "168h",
			MaxCacheBytes:  1073741824,
			BrotliQuality:  5,
			GzipLevel:      5,
		},
	}
}
