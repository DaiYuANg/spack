package media

import (
	"strings"

	"github.com/arcgolabs/collectionx"
	"github.com/daiyuang/spack/internal/constant"
)

var textLikeMediaTypes = collectionx.NewOrderedSet[string](
	string(constant.ApplicationJavascript),
	string(constant.XJavascript),
	string(constant.JSON),
	string(constant.ManifestJSON),
	string(constant.XML),
	string(constant.XHTML),
	string(constant.Svg),
)

var compressibleNonTextMediaTypes = collectionx.NewOrderedSet[string](
	string(constant.Wasm),
)

func NormalizeMediaType(mediaType string) string {
	return strings.ToLower(strings.TrimSpace(mediaType))
}

func IsTextLikeMediaType(mediaType string) bool {
	normalized := NormalizeMediaType(mediaType)
	switch {
	case strings.HasPrefix(normalized, "text/"):
		return true
	case textLikeMediaTypes.Contains(normalized):
		return true
	default:
		return !IsImageMediaType(normalized) && strings.Contains(normalized, "json")
	}
}

func IsNonCompressibleMediaType(mediaType string) bool {
	normalized := NormalizeMediaType(mediaType)
	if strings.HasPrefix(normalized, "image/") && normalized != string(constant.Svg) {
		return true
	}
	if strings.HasPrefix(normalized, "audio/") || strings.HasPrefix(normalized, "video/") {
		return true
	}
	return strings.Contains(normalized, "zip") || strings.Contains(normalized, "gzip")
}

func IsCompressibleMediaType(mediaType string) bool {
	normalized := NormalizeMediaType(mediaType)
	switch {
	case normalized == "":
		return false
	case IsNonCompressibleMediaType(normalized):
		return false
	case IsTextLikeMediaType(normalized):
		return true
	default:
		return compressibleNonTextMediaTypes.Contains(normalized)
	}
}
