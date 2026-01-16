package config

func defaultConfig() Config {
	return Config{
		Http: Http{
			Port:      80,
			LowMemory: true,
		},
		Spa: Spa{
			Path:             "/",
			Fallback:         "index.html",
			Preload:          false,
			NotFoundFallback: true,
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
		Debug: Debug{
			Enable:      true,
			PprofPrefix: "/pprof",
			LivePort:    8080,
		},
	}
}
