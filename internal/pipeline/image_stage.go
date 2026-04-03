package pipeline

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"golang.org/x/image/draw"
)

type imageStage struct {
	cfg     *config.Image
	store   artifact.Store
	catalog catalog.Catalog
}

type imageStageIn struct {
	Config  *config.Image
	Store   artifact.Store
	Catalog catalog.Catalog
}

func newImageStage(in imageStageIn) Stage {
	return &imageStage{
		cfg:     in.Config,
		store:   in.Store,
		catalog: in.Catalog,
	}
}

func newImageStageFromDeps(cfg *config.Image, store artifact.Store, cat catalog.Catalog) *imageStage {
	return newImageStage(imageStageIn{
		Config:  cfg,
		Store:   store,
		Catalog: cat,
	}).(*imageStage)
}

func (s *imageStage) Name() string {
	return "image"
}

func (s *imageStage) Plan(asset *catalog.Asset, request Request) []Task {
	if !s.cfg.Enable || !isResizableImage(asset) {
		return nil
	}

	formats := normalizeImageFormats(request.PreferredFormats)
	if len(formats) == 0 {
		formats = []string{imageFormat(asset.MediaType)}
	}

	widths := request.PreferredWidths
	if len(widths) == 0 {
		if len(request.PreferredFormats) > 0 {
			widths = []int{0}
		} else {
			widths = s.cfg.ParsedWidths()
		}
	}
	if len(widths) == 0 {
		return nil
	}

	existing := s.catalog.ListVariants(asset.Path)
	tasks := make([]Task, 0, len(widths)*len(formats))
	for _, format := range formats {
		for _, width := range widths {
			if width == 0 && format == imageFormat(asset.MediaType) {
				continue
			}
			if width < 0 || hasImageVariant(existing, asset.SourceHash, width, format) {
				continue
			}

			tasks = append(tasks, Task{
				AssetPath: asset.Path,
				Format:    format,
				Width:     width,
			})
		}
	}
	return tasks
}

func (s *imageStage) Execute(task Task, asset *catalog.Asset) (*catalog.Variant, error) {
	targetFormat := task.Format
	if targetFormat == "" {
		targetFormat = imageFormat(asset.MediaType)
	}
	if task.Width < 0 {
		return nil, nil
	}
	if task.Width == 0 && targetFormat == imageFormat(asset.MediaType) {
		return nil, nil
	}

	file, err := os.Open(asset.FullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	srcImage, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := srcImage.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth <= 0 || srcHeight <= 0 {
		return nil, nil
	}

	outputImage := srcImage
	outputWidth := srcWidth
	if task.Width > 0 && task.Width < srcWidth {
		targetHeight := max(1, srcHeight*task.Width/srcWidth)
		dst := image.NewRGBA(image.Rect(0, 0, task.Width, targetHeight))
		draw.CatmullRom.Scale(dst, dst.Bounds(), srcImage, bounds, draw.Over, nil)
		outputImage = dst
		outputWidth = task.Width
	}

	payload, ext, mediaType, err := encodeImage(outputImage, targetFormat, clampJPEGQuality(s.cfg.JPEGQuality))
	if err != nil {
		return nil, err
	}
	if outputWidth == srcWidth && mediaType == asset.MediaType && int64(len(payload)) >= asset.Size {
		return nil, nil
	}

	targetPath := s.store.PathFor(asset.Path, asset.SourceHash, "image", imageVariantSuffix(outputWidth, targetFormat, ext))
	if err := s.store.Write(targetPath, payload); err != nil {
		return nil, err
	}

	return &catalog.Variant{
		ID:           imageVariantID(asset.Path, outputWidth, targetFormat),
		AssetPath:    asset.Path,
		ArtifactPath: targetPath,
		Size:         int64(len(payload)),
		MediaType:    mediaType,
		SourceHash:   asset.SourceHash,
		ETag:         imageVariantETag(asset.SourceHash, outputWidth, targetFormat),
		Format:       targetFormat,
		Width:        outputWidth,
		Metadata: map[string]string{
			"stage": "image",
		},
	}, nil
}

func isResizableImage(asset *catalog.Asset) bool {
	switch strings.ToLower(strings.TrimSpace(asset.MediaType)) {
	case "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}

func hasImageVariant(variants []*catalog.Variant, sourceHash string, width int, format string) bool {
	for _, variant := range variants {
		if variant.Width != width {
			continue
		}
		if format != "" && variant.Format != format {
			continue
		}
		if sourceHash != "" && variant.SourceHash != "" && variant.SourceHash != sourceHash {
			continue
		}
		if variant.ArtifactPath == "" {
			continue
		}
		if _, err := os.Stat(variant.ArtifactPath); err != nil {
			continue
		}
		return true
	}
	return false
}

func encodeImage(img image.Image, format string, jpegQuality int) ([]byte, string, string, error) {
	var buf bytes.Buffer
	switch normalizeImageFormat(format) {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, "", "", err
		}
		return buf.Bytes(), ".jpg", "image/jpeg", nil
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, "", "", err
		}
		return buf.Bytes(), ".png", "image/png", nil
	default:
		return nil, "", "", fmt.Errorf("unsupported image format: %s", format)
	}
}

func clampJPEGQuality(quality int) int {
	if quality < 1 {
		return 1
	}
	if quality > 100 {
		return 100
	}
	return quality
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

func normalizeImageFormats(formats []string) []string {
	if len(formats) == 0 {
		return nil
	}

	seen := collectionset.NewSetWithCapacity[string](len(formats))
	out := make([]string, 0, len(formats))
	for _, format := range formats {
		normalized := normalizeImageFormat(format)
		if normalized == "" {
			continue
		}
		if seen.Contains(normalized) {
			continue
		}
		seen.Add(normalized)
		out = append(out, normalized)
	}
	return out
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

func imageVariantSuffix(width int, format string, ext string) string {
	parts := make([]string, 0, 2)
	if width > 0 {
		parts = append(parts, fmt.Sprintf("w%d", width))
	}
	if format != "" {
		parts = append(parts, "f"+format)
	}
	if len(parts) == 0 {
		return ext
	}
	return "." + strings.Join(parts, ".") + ext
}

func imageVariantID(assetPath string, width int, format string) string {
	parts := []string{assetPath}
	if width > 0 {
		parts = append(parts, fmt.Sprintf("width=%d", width))
	}
	if format != "" {
		parts = append(parts, "format="+format)
	}
	return strings.Join(parts, "|")
}

func imageVariantETag(sourceHash string, width int, format string) string {
	parts := []string{sourceHash}
	if width > 0 {
		parts = append(parts, fmt.Sprintf("w%d", width))
	}
	if format != "" {
		parts = append(parts, format)
	}
	return "\"" + strings.Join(parts, "-") + "\""
}
