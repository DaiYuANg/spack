package pipeline

import (
	"fmt"
	"os"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/media"
)

func (s *imageStage) planFormats(asset *catalog.Asset, request Request) collectionx.List[string] {
	var supported collectionx.List[string]
	if s.engine != nil {
		supported = normalizeImageFormats(s.engine.SupportedTargetFormats())
	}

	formats := filterSupportedImageFormats(request.PreferredFormats, supported)
	if formats != nil && !formats.IsEmpty() {
		return formats
	}

	sourceFormat := media.ImageFormat(asset.MediaType)
	defaultFormats := filterSupportedImageFormats(s.cfg.ParsedFormats(), supported)
	if sourceFormat != "" {
		if defaultFormats == nil {
			defaultFormats = collectionx.NewList[string]()
		}
		defaultFormats.Add(sourceFormat)
	}
	return filterSupportedImageFormats(defaultFormats, supported)
}

func (s *imageStage) planWidths(asset *catalog.Asset, request Request, formats collectionx.List[string]) collectionx.List[int] {
	if request.PreferredWidths != nil && !request.PreferredWidths.IsEmpty() {
		return request.PreferredWidths
	}
	if request.PreferredFormats != nil && request.PreferredFormats.Len() > 0 {
		return collectionx.NewList[int](0)
	}

	widths := collectionx.NewList[int](s.cfg.ParsedWidths().Values()...)
	if shouldPlanOriginalFormatVariants(formats, media.ImageFormat(asset.MediaType)) {
		widths.Add(0)
	}
	if widths.IsEmpty() {
		return widths
	}

	widths.Sort(func(left, right int) int {
		return left - right
	})
	return collectionx.NewList[int](collectionx.NewOrderedSet[int](widths.Values()...).Values()...)
}

func shouldPlanOriginalFormatVariants(formats collectionx.List[string], sourceFormat string) bool {
	if formats == nil || formats.IsEmpty() {
		return false
	}

	shouldPlan := false
	formats.Range(func(_ int, format string) bool {
		if format != "" && format != sourceFormat {
			shouldPlan = true
			return false
		}
		return true
	})
	return shouldPlan
}

func (s *imageStage) planTasks(asset *catalog.Asset, formats collectionx.List[string], widths collectionx.List[int]) collectionx.List[Task] {
	if formats == nil || widths == nil {
		return nil
	}
	return collectionx.FlatMapList[string, Task](formats, func(_ int, format string) []Task {
		return collectionx.FilterMapList[int, Task](widths, func(_ int, width int) (Task, bool) {
			if !shouldCreateImageTask(asset, s.catalog, width, format) {
				return Task{}, false
			}
			return Task{
				AssetPath: asset.Path,
				Format:    format,
				Width:     width,
			}, true
		}).Values()
	})
}

func shouldCreateImageTask(asset *catalog.Asset, cat catalog.Catalog, width int, format string) bool {
	if width < 0 {
		return false
	}
	if width == 0 && format == media.ImageFormat(asset.MediaType) {
		return false
	}
	variant, ok := cat.FindImageVariant(asset.Path, format, width)
	if !ok {
		return true
	}
	return !hasImageVariant(variant, asset.SourceHash, width, format)
}

func resolveTargetFormat(task Task, asset *catalog.Asset) (string, error) {
	targetFormat := task.Format
	if targetFormat == "" {
		targetFormat = media.ImageFormat(asset.MediaType)
	}
	if task.Width < 0 {
		return "", ErrVariantSkipped
	}
	if task.Width == 0 && targetFormat == media.ImageFormat(asset.MediaType) {
		return "", ErrVariantSkipped
	}
	if targetFormat == "" {
		return "", fmt.Errorf("unsupported image format: %s", task.Format)
	}
	return targetFormat, nil
}

func shouldSkipImageArtifact(asset *catalog.Asset, result imageGenerateResult) bool {
	return result.Width == result.SourceWidth && result.MediaType == asset.MediaType && int64(len(result.Payload)) >= asset.Size
}

func hasImageVariant(variant *catalog.Variant, sourceHash string, width int, format string) bool {
	return isMatchingImageVariant(variant, sourceHash, width, format)
}

func isMatchingImageVariant(variant *catalog.Variant, sourceHash string, width int, format string) bool {
	if variant.Width != width {
		return false
	}
	if format != "" && variant.Format != format {
		return false
	}
	if sourceHash != "" && variant.SourceHash != "" && variant.SourceHash != sourceHash {
		return false
	}
	if variant.ArtifactPath == "" {
		return false
	}
	_, err := os.Stat(variant.ArtifactPath)
	return err == nil
}

func filterSupportedImageFormats(formats, supported collectionx.List[string]) collectionx.List[string] {
	normalized := normalizeImageFormats(formats)
	if normalized == nil || normalized.IsEmpty() || supported == nil || supported.IsEmpty() {
		return normalized
	}

	return collectionx.FilterMapList[string, string](normalized, func(_ int, format string) (string, bool) {
		_, ok := supported.FirstWhere(func(_ int, candidate string) bool {
			return candidate == format
		}).Get()
		return format, ok
	})
}
