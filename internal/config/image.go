package config

import (
	"slices"
	"strconv"
	"strings"
)

type Image struct {
	Enable      bool   `koanf:"enable"`
	Widths      string `koanf:"widths"`
	JPEGQuality int    `koanf:"jpeg_quality"`
}

func (i Image) ParsedWidths() []int {
	if strings.TrimSpace(i.Widths) == "" {
		return nil
	}

	parts := strings.Split(i.Widths, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		width, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || width <= 0 {
			continue
		}
		out = append(out, width)
	}
	slices.Sort(out)

	unique := out[:0]
	var last int
	for index, width := range out {
		if index == 0 || width != last {
			unique = append(unique, width)
			last = width
		}
	}
	return unique
}
