package pipeline

import (
	"bytes"
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/anthonynsimon/bild/imgio"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/mediax"
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

func newImageStage(in imageStageIn) *imageStage {
	return &imageStage{
		cfg:     in.Config,
		store:   in.Store,
		catalog: in.Catalog,
	}
}

func newImageStageFromDeps(cfg *config.Image, store artifact.Store, cat catalog.Catalog) *imageStage {
	return newImageStage(imageStageIn{Config: cfg, Store: store, Catalog: cat})
}

func (s *imageStage) Name() string {
	return "image"
}

func (s *imageStage) Plan(asset *catalog.Asset, request Request) []Task {
	if !s.cfg.Enable || !isResizableImage(asset) {
		return nil
	}

	formats := s.planFormats(asset, request)
	widths := s.planWidths(request)
	if widths.IsEmpty() {
		return nil
	}

	return s.planTasks(asset, formats, widths)
}

func (s *imageStage) Execute(task Task, asset *catalog.Asset) (*catalog.Variant, error) {
	targetFormat, err := resolveTargetFormat(task, asset)
	if err != nil {
		return nil, err
	}

	srcImage, srcWidth, srcHeight, err := loadSourceImage(asset.FullPath)
	if err != nil {
		return nil, err
	}
	if srcWidth <= 0 || srcHeight <= 0 {
		return nil, ErrVariantSkipped
	}

	outputImage, outputWidth := resizeImage(srcImage, srcWidth, srcHeight, task.Width)

	payload, ext, mediaType, err := encodeImage(outputImage, targetFormat, clampJPEGQuality(s.cfg.JPEGQuality))
	if err != nil {
		return nil, fmt.Errorf("encode image artifact: %w", err)
	}
	if shouldSkipImageArtifact(asset, srcWidth, outputWidth, mediaType, len(payload)) {
		return nil, ErrVariantSkipped
	}

	targetPath := s.store.PathFor(asset.Path, asset.SourceHash, "image", imageVariantSuffix(outputWidth, targetFormat, ext))
	if err := s.store.Write(targetPath, payload); err != nil {
		return nil, fmt.Errorf("write image artifact: %w", err)
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
			"stage":      "image",
			"mtime_unix": strconv.FormatInt(time.Now().Unix(), 10),
		},
	}, nil
}

func isResizableImage(asset *catalog.Asset) bool {
	return mediax.ImageFormat(asset.MediaType) != ""
}

func encodeImage(img image.Image, format string, jpegQuality int) ([]byte, string, string, error) {
	var buf bytes.Buffer
	var encoder imgio.Encoder
	switch mediax.NormalizeImageFormat(format) {
	case "jpeg":
		encoder = imgio.JPEGEncoder(jpegQuality)
		if err := encoder(&buf, img); err != nil {
			return nil, "", "", fmt.Errorf("encode jpeg image: %w", err)
		}
		return buf.Bytes(), ".jpg", "image/jpeg", nil
	case "png":
		encoder = imgio.PNGEncoder()
		if err := encoder(&buf, img); err != nil {
			return nil, "", "", fmt.Errorf("encode png image: %w", err)
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

func normalizeImageFormats(formats collectionx.List[string]) collectionx.List[string] {
	return mediax.NormalizeImageFormats(formats)
}

func imageVariantSuffix(width int, format, ext string) string {
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
