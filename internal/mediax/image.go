// Package mediax centralizes media-type and image-format normalization helpers.
package mediax

import (
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func NormalizeImageFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	default:
		return ""
	}
}

func ImageFormat(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	default:
		return ""
	}
}

func IsImageMediaType(mediaType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/")
}

func NormalizeImageFormats(formats collectionx.List[string]) collectionx.List[string] {
	if formats.IsEmpty() {
		return collectionx.NewList[string]()
	}

	normalized := collectionx.FilterMapList(formats, func(_ int, format string) (string, bool) {
		value := NormalizeImageFormat(format)
		return value, value != ""
	})
	return collectionx.NewList(collectionx.NewOrderedSet(normalized.Values()...).Values()...)
}
