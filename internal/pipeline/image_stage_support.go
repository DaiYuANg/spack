package pipeline

import (
	"fmt"
	"image"
	"os"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"golang.org/x/image/draw"
)

func (s *imageStage) planFormats(asset *catalog.Asset, request Request) collectionx.List[string] {
	formats := normalizeImageFormats(request.PreferredFormats)
	if !formats.IsEmpty() {
		return formats
	}
	return collectionx.NewList(imageFormat(asset.MediaType))
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

func (s *imageStage) planTasks(asset *catalog.Asset, formats collectionx.List[string], widths collectionx.List[int]) []Task {
	existing := s.catalog.ListVariants(asset.Path)
	tasks := make([]Task, 0, widths.Len()*formats.Len())
	formats.Range(func(_ int, format string) bool {
		widths.Range(func(_ int, width int) bool {
			if shouldCreateImageTask(asset, existing, width, format) {
				tasks = append(tasks, Task{
					AssetPath: asset.Path,
					Format:    format,
					Width:     width,
				})
			}
			return true
		})
		return true
	})
	return tasks
}

func shouldCreateImageTask(asset *catalog.Asset, variants collectionx.List[*catalog.Variant], width int, format string) bool {
	if width < 0 {
		return false
	}
	if width == 0 && format == imageFormat(asset.MediaType) {
		return false
	}
	return !hasImageVariant(variants, asset.SourceHash, width, format)
}

func resolveTargetFormat(task Task, asset *catalog.Asset) (string, error) {
	targetFormat := task.Format
	if targetFormat == "" {
		targetFormat = imageFormat(asset.MediaType)
	}
	if task.Width < 0 {
		return "", ErrVariantSkipped
	}
	if task.Width == 0 && targetFormat == imageFormat(asset.MediaType) {
		return "", ErrVariantSkipped
	}
	return targetFormat, nil
}

func loadSourceImage(path string) (image.Image, int, int, error) {
	// #nosec G304 -- image paths come from the scanned local asset tree.
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("open source image: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			return
		}
	}()

	srcImage, _, err := image.Decode(file)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode source image: %w", err)
	}

	bounds := srcImage.Bounds()
	return srcImage, bounds.Dx(), bounds.Dy(), nil
}

func resizeImage(srcImage image.Image, srcWidth, srcHeight, targetWidth int) (image.Image, int) {
	if targetWidth <= 0 || targetWidth >= srcWidth {
		return srcImage, srcWidth
	}

	targetHeight := max(1, srcHeight*targetWidth/srcWidth)
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), srcImage, srcImage.Bounds(), draw.Over, nil)
	return dst, targetWidth
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
