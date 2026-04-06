package config

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/validation"
)

type Image struct {
	Enable      bool   `koanf:"enable"`
	Widths      string `koanf:"widths"       validate:"omitempty,spack_widths"`
	JPEGQuality int    `koanf:"jpeg_quality" validate:"gte=1,lte=100"`
}

func (i Image) ParsedWidths() collectionx.List[int] {
	return validation.ParseWidths(i.Widths)
}
