package resolver

import (
	"os"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/mediax"
	"github.com/samber/lo"
)

func uniqueStrings(values collectionx.List[string]) collectionx.List[string] {
	filtered := collectionx.FilterMapList(values, func(_ int, value string) (string, bool) {
		return value, value != ""
	})
	return collectionx.NewList(collectionx.NewOrderedSet(filtered.Values()...).Values()...)
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
	return lo.FindOrElse(values, "", func(value string) bool {
		return strings.TrimSpace(value) != ""
	})
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
	if !mediax.IsImageMediaType(sourceMediaType) {
		return collectionx.NewList[string]()
	}
	return parseAcceptImageFormats(acceptHeader, mediax.ImageFormat(sourceMediaType))
}
