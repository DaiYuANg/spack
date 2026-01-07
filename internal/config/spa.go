package config

type Spa struct {
	Path string `koanf:"path"`
	//Serve scanner spa config
	Static string `koanf:"static"`
	//default load file config like nginx try file
	Fallback string `koanf:"fallback"`
	Preload  bool   `koanf:"preload"`
}
