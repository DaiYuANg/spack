package pipeline

import (
	"fmt"
	"image"
	"os"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/media"
	"github.com/samber/lo"
)

func (s *imageStage) planFormats(asset *catalog.Asset, request Request) collectionx.List[string] {
	formats := normalizeImageFormats(request.PreferredFormats)
	if !formats.IsEmpty() {
		return formats
	}
	return collectionx.NewList(media.ImageFormat(asset.MediaType))
}

func (s *imageStage) planWidths(request Request) collectionx.List[int] {
	if !request.PreferredWidths.IsEmpty() {
		return request.PreferredWidths
	}
	if request.PreferredFormats.Len() > 0 {
		return collectionx.NewList(0)
	}
	return s.cfg.ParsedWidths()
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
	return targetFormat, nil
}

func loadSourceImage(path string) (image.Image, int, int, error) {
	srcImage, err := imgio.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("open source image: %w", err)
	}

	bounds := srcImage.Bounds()
	return srcImage, bounds.Dx(), bounds.Dy(), nil
}

func resizeImage(srcImage image.Image, srcWidth, srcHeight, targetWidth int) (image.Image, int) {
	if targetWidth <= 0 || targetWidth >= srcWidth {
		return srcImage, srcWidth
	}

	targetHeight := max(1, srcHeight*targetWidth/srcWidth)
	return transform.Resize(srcImage, targetWidth, targetHeight, transform.CatmullRom), targetWidth
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
