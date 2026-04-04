package config

import (
	"github.com/DaiYuANg/arcgo/configx"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/spf13/pflag"
)

// LoadOptions controls which external config sources are consulted in addition
// to the built-in defaults, dotenv files, and environment variables.
type LoadOptions struct {
	Files   []string
	FlagSet *pflag.FlagSet
}

func (o LoadOptions) configxOptions() []configx.Option {
	options := []configx.Option{
		configx.WithEnvPrefix(constant.EnvPrefix),
		configx.WithIgnoreDotenvError(true),
		configx.WithDotenv(),
	}
	if len(o.Files) > 0 {
		options = append(options, configx.WithFiles(o.Files...))
	}
	if o.FlagSet != nil {
		options = append(options, configx.WithFlagSet(o.FlagSet))
	}
	return options
}
