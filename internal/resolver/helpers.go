package resolver

import (
	"os"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
)

func uniqueStrings(values collectionx.List[string]) collectionx.List[string] {
	ordered := collectionx.NewOrderedSetWithCapacity[string](values.Len())
	values.Each(func(_ int, value string) {
		if value == "" {
			return
		}
		ordered.Add(value)
	})
	return collectionx.NewList(ordered.Values()...)
}

func isUsableVariant(variant *catalog.Variant, assetSourceHash string) bool {
	if variant == nil || strings.TrimSpace(variant.ArtifactPath) == "" {
		return false
	}
	if assetSourceHash != "" && variant.SourceHash != "" && variant.SourceHash != assetSourceHash {
		return false
	}
	if _, err := os.Stat(variant.ArtifactPath); err != nil {
		return false
	}
	return true
}

func variantFormat(variant *catalog.Variant, sourceFormat string) string {
	if variant == nil || strings.TrimSpace(variant.Format) == "" {
		return sourceFormat
	}
	return variant.Format
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func preferredWidths(width int) collectionx.List[int] {
	if width <= 0 {
		return collectionx.NewList[int]()
	}
	return collectionx.NewList(width)
}

func preferredImageFormats(acceptHeader, explicitFormat, sourceMediaType string) collectionx.List[string] {
	if explicitFormat != "" {
		return collectionx.NewList(explicitFormat)
	}
	if !isImageMediaType(sourceMediaType) {
		return collectionx.NewList[string]()
	}
	return parseAcceptImageFormats(acceptHeader, imageFormat(sourceMediaType))
}

func normalizeImageFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	default:
		return ""
	}
}

func imageFormat(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	default:
		return ""
	}
}

func isImageMediaType(mediaType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/")
}
