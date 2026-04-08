package config

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/configx"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/pflag"
)

// LoadOptions controls which external config sources are consulted in addition
// to the built-in defaults, dotenv files, and environment variables.
type LoadOptions struct {
	Files   []string
	FlagSet *pflag.FlagSet
}

func (o LoadOptions) configxOptions(validate *validator.Validate) []configx.Option {
	options := collectionx.NewList[configx.Option](
		configx.WithEnvPrefix(constant.EnvPrefix),
		configx.WithIgnoreDotenvError(true),
		configx.WithDotenv(),
		configx.WithValidator(validate),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if len(o.Files) > 0 {
		options.Add(configx.WithFiles(o.Files...))
	}
	if o.FlagSet != nil {
		options.Add(configx.WithFlagSet(o.FlagSet))
	}
	return options.Values()
}
