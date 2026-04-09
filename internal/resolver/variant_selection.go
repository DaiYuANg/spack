package resolver

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
)

func pickImageVariantForFormat(
	variants collectionx.List[*catalog.Variant],
	usable variantUsabilityCache,
	sourceHash string,
	format string,
	sourceFormat string,
	width int,
) *catalog.Variant {
	if width <= 0 {
		return findZeroWidthImageVariant(variants, usable, sourceHash, format, sourceFormat)
	}
	return findClosestWidthImageVariant(variants, usable, sourceHash, format, sourceFormat, width)
}

func findZeroWidthImageVariant(
	variants collectionx.List[*catalog.Variant],
	usable variantUsabilityCache,
	sourceHash string,
	format string,
	sourceFormat string,
) *catalog.Variant {
	var picked *catalog.Variant
	variants.Range(func(_ int, candidate *catalog.Variant) bool {
		if candidate.Width != 0 || !matchesImageVariant(candidate, usable, sourceHash, format, sourceFormat) {
			return true
		}
		picked = candidate
		return false
	})
	return picked
}

func findClosestWidthImageVariant(
	variants collectionx.List[*catalog.Variant],
	usable variantUsabilityCache,
	sourceHash string,
	format string,
	sourceFormat string,
	width int,
) *catalog.Variant {
	var smallestAbove *catalog.Variant
	var largestBelow *catalog.Variant
	variants.Range(func(_ int, candidate *catalog.Variant) bool {
		if !matchesImageVariant(candidate, usable, sourceHash, format, sourceFormat) {
			return true
		}
		smallestAbove, largestBelow = updateWidthMatches(candidate, width, smallestAbove, largestBelow)
		return true
	})
	if smallestAbove != nil {
		return smallestAbove
	}
	return largestBelow
}

func updateWidthMatches(
	candidate *catalog.Variant,
	width int,
	smallestAbove *catalog.Variant,
	largestBelow *catalog.Variant,
) (*catalog.Variant, *catalog.Variant) {
	if candidate.Width >= width {
		if smallestAbove == nil || candidate.Width < smallestAbove.Width {
			smallestAbove = candidate
		}
		return smallestAbove, largestBelow
	}
	if largestBelow == nil || candidate.Width > largestBelow.Width {
		largestBelow = candidate
	}
	return smallestAbove, largestBelow
}

func matchesImageVariant(
	candidate *catalog.Variant,
	usable variantUsabilityCache,
	sourceHash string,
	format string,
	sourceFormat string,
) bool {
	if candidate == nil || (candidate.Width <= 0 && candidate.Format == "") {
		return false
	}
	if !usable.IsUsable(candidate, sourceHash) {
		return false
	}
	return format == "" || variantFormat(candidate, sourceFormat) == format
}
