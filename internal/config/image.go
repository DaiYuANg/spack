package config

import (
	"strings"

	"github.com/arcgolabs/collectionx"
	"github.com/daiyuang/spack/internal/validation"
)

type Image struct {
	Enable      bool   `koanf:"enable"`
	Widths      string `koanf:"widths"       validate:"omitempty,spack_widths"`
	Formats     string `koanf:"formats"`
	JPEGQuality int    `koanf:"jpeg_quality" validate:"gte=1,lte=100"`
}

func (i Image) ParsedWidths() collectionx.List[int] {
	return validation.ParseWidths(i.Widths)
}

func (i Image) ParsedFormats() collectionx.List[string] {
	if strings.TrimSpace(i.Formats) == "" {
		return collectionx.NewList[string]()
	}
	return collectionx.NewList[string](strings.Split(i.Formats, ",")...)
}
