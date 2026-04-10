package resolver

import (
	"os"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/media"
	"github.com/samber/lo"
)

var supportedImageFormats = media.SupportedImageFormats()

type variantViewCatalog interface {
	ListVariantsView(assetPath string) collectionx.List[*catalog.Variant]
}

type assetViewCatalog interface {
	FindAssetView(assetPath string) (*catalog.Asset, bool)
}

func findAssetForRead(cat catalog.Catalog, assetPath string) (*catalog.Asset, bool) {
	if fastCat, ok := cat.(assetViewCatalog); ok {
		return fastCat.FindAssetView(assetPath)
	}
	return cat.FindAsset(assetPath)
}

func listVariantsForRead(cat catalog.Catalog, assetPath string) collectionx.List[*catalog.Variant] {
	if fastCat, ok := cat.(variantViewCatalog); ok {
		return fastCat.ListVariantsView(assetPath)
	}
	return cat.ListVariants(assetPath)
}

type variantUsabilityCache map[string]bool

func newVariantUsabilityCache() variantUsabilityCache {
	return make(variantUsabilityCache, 4)
}

func (cache variantUsabilityCache) IsUsable(variant *catalog.Variant, assetSourceHash string) bool {
	if variant == nil {
		return false
	}

	key := variant.ID
	if key == "" {
		key = variant.ArtifactPath
	}
	if usable, ok := cache[key]; ok {
		return usable
	}

	usable := isUsableVariant(variant, assetSourceHash)
	cache[key] = usable
	return usable
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
	return firstNonEmpty(lo.Ternary(variant != nil, variant.Format, ""), sourceFormat)
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

func preferredImageFormats(
	acceptHeader,
	explicitFormat,
	sourceMediaType string,
) collectionx.List[string] {
	if explicitFormat != "" {
		return collectionx.NewList(explicitFormat)
	}
	if !media.IsImageMediaType(sourceMediaType) {
		return collectionx.NewList[string]()
	}
	return parseAcceptImageFormats(acceptHeader, media.ImageFormat(sourceMediaType), supportedImageFormats)
}
