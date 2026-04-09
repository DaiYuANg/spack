package pipeline

import (
	"fmt"
	"os"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/media"
	"github.com/samber/lo"
)

func (s *imageStage) planFormats(asset *catalog.Asset, request Request) collectionx.List[string] {
	supported := collectionx.NewList[string]()
	if s.engine != nil {
		supported = normalizeImageFormats(s.engine.SupportedTargetFormats())
	}

	formats := filterSupportedImageFormats(request.PreferredFormats, supported)
	if !formats.IsEmpty() {
		return formats
	}

	sourceFormat := media.ImageFormat(asset.MediaType)
	defaultFormats := filterSupportedImageFormats(s.cfg.ParsedFormats(), supported)
	if sourceFormat != "" {
		defaultFormats.Add(sourceFormat)
	}
	return filterSupportedImageFormats(defaultFormats, supported)
}

func (s *imageStage) planWidths(asset *catalog.Asset, request Request, formats collectionx.List[string]) collectionx.List[int] {
	if !request.PreferredWidths.IsEmpty() {
		return request.PreferredWidths
	}
	if request.PreferredFormats.Len() > 0 {
		return collectionx.NewList(0)
	}

	widths := collectionx.NewList(s.cfg.ParsedWidths().Values()...)
	if shouldPlanOriginalFormatVariants(formats, media.ImageFormat(asset.MediaType)) {
		widths.Add(0)
	}
	if widths.IsEmpty() {
		return widths
	}

	widths.Sort(func(left, right int) int {
		return left - right
	})
	return collectionx.NewList(collectionx.NewOrderedSet(widths.Values()...).Values()...)
}

func shouldPlanOriginalFormatVariants(formats collectionx.List[string], sourceFormat string) bool {
	if formats.IsEmpty() {
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
	existing := s.catalog.ListVariants(asset.Path)
	return collectionx.NewList(lo.FlatMap(formats.Values(), func(format string, _ int) []Task {
		return lo.FilterMap(widths.Values(), func(width int, _ int) (Task, bool) {
			if !shouldCreateImageTask(asset, existing, width, format) {
				return Task{}, false
			}
			return Task{
				AssetPath: asset.Path,
				Format:    format,
				Width:     width,
			}, true
		})
	})...)
}

func shouldCreateImageTask(asset *catalog.Asset, variants collectionx.List[*catalog.Variant], width int, format string) bool {
	if width < 0 {
		return false
	}
	if width == 0 && format == media.ImageFormat(asset.MediaType) {
		return false
	}
	return !hasImageVariant(variants, asset.SourceHash, width, format)
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

func shouldSkipImageArtifact(asset *catalog.Asset, srcWidth, outputWidth int, mediaType string, payloadSize int) bool {
	return outputWidth == srcWidth && mediaType == asset.MediaType && int64(payloadSize) >= asset.Size
}

func hasImageVariant(variants collectionx.List[*catalog.Variant], sourceHash string, width int, format string) bool {
	_, ok := variants.FirstWhere(func(_ int, variant *catalog.Variant) bool {
		return isMatchingImageVariant(variant, sourceHash, width, format)
	}).Get()
	return ok
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

func filterSupportedImageFormats(formats collectionx.List[string], supported collectionx.List[string]) collectionx.List[string] {
	normalized := normalizeImageFormats(formats)
	if normalized.IsEmpty() || supported.IsEmpty() {
		return normalized
	}

	return collectionx.FilterMapList(normalized, func(_ int, format string) (string, bool) {
		_, ok := supported.FirstWhere(func(_ int, candidate string) bool {
			return candidate == format
		}).Get()
		return format, ok
	})
}
