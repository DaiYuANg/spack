package config

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type Image struct {
	Enable      bool   `koanf:"enable"`
	Widths      string `koanf:"widths"`
	JPEGQuality int    `koanf:"jpeg_quality"`
}

func (i Image) ParsedWidths() collectionx.List[int] {
	if strings.TrimSpace(i.Widths) == "" {
		return collectionx.NewList[int]()
	}

	widths := collectionx.FilterMapList(collectionx.NewList(strings.Split(i.Widths, ",")...), func(_ int, part string) (int, bool) {
		width, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || width <= 0 {
			return 0, false
		}
		return width, true
	})
	if widths.IsEmpty() {
		return widths
	}

	widths.Sort(cmp.Compare[int])
	return collectionx.NewList(collectionx.NewOrderedSet(widths.Values()...).Values()...)
}
