package config

import (
	"os"
	"path/filepath"
)

func defaultConfig() Config {
	return Config{
		Http: Http{
			Port:      80,
			LowMemory: true,
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
		Processor: Processor{
			Enable: false,
		},
		Metrics: Metrics{Prefix: "/premetheus"},
		Debug: Debug{
			Enable:      true,
			PprofPrefix: "/pprof",
			LivePort:    8080,
		},
		Compression: Compression{
			Enable:        true,
			CacheDir:      filepath.Join(os.TempDir(), "spack-cache"),
			MinSize:       1024,
			Workers:       2,
			BrotliQuality: 5,
			GzipLevel:     5,
		},
	}
}
