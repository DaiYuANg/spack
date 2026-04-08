package pipeline

import (
	"bytes"
	"fmt"
	"image"
	"strconv"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/media"
	"github.com/samber/oops"
)

type imageStage struct {
	cfg     *config.Image
	store   artifact.Store
	catalog catalog.Catalog
}

func newImageStage(cfg *config.Image, store artifact.Store, cat catalog.Catalog) *imageStage {
	return &imageStage{
		cfg:     cfg,
		store:   store,
		catalog: cat,
	}
}

func (s *imageStage) Name() string {
	return "image"
}

func (s *imageStage) Plan(asset *catalog.Asset, request Request) collectionx.List[Task] {
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
		return nil, oops.Wrap(fmt.Errorf("write image artifact: %w", err))
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
		Metadata: collectionx.NewMapFrom(map[string]string{
			"stage":      "image",
			"mtime_unix": strconv.FormatInt(time.Now().Unix(), 10),
		}),
	}, nil
}

func isResizableImage(asset *catalog.Asset) bool {
	_, ok := media.LookupImageCapabilityByMediaType(asset.MediaType)
	return ok
}

func encodeImage(img image.Image, format string, jpegQuality int) ([]byte, string, string, error) {
	var buf bytes.Buffer
	capability, ok := media.LookupImageCapability(media.NormalizeImageFormat(format))
	if !ok {
		return nil, "", "", fmt.Errorf("unsupported image format: %s", format)
	}

	encoder := capability.Encoder(jpegQuality)
	if err := encoder(&buf, img); err != nil {
		return nil, "", "", fmt.Errorf("encode %s image: %w", capability.Name, err)
	}
	return buf.Bytes(), capability.Extension, capability.MediaType, nil
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
	return media.NormalizeImageFormats(formats)
}

func imageVariantSuffix(width int, format, ext string) string {
	parts := collectionx.NewList[string]()
	if width > 0 {
		parts.Add(fmt.Sprintf("w%d", width))
	}
	if format != "" {
		parts.Add("f" + format)
	}
	if parts.IsEmpty() {
		return ext
	}
	return "." + parts.Join(".") + ext
}

func imageVariantID(assetPath string, width int, format string) string {
	parts := collectionx.NewList(assetPath)
	if width > 0 {
		parts.Add(fmt.Sprintf("width=%d", width))
	}
	if format != "" {
		parts.Add("format=" + format)
	}
	return parts.Join("|")
}

func imageVariantETag(sourceHash string, width int, format string) string {
	parts := collectionx.NewList(sourceHash)
	if width > 0 {
		parts.Add(fmt.Sprintf("w%d", width))
	}
	if format != "" {
		parts.Add(format)
	}
	return "\"" + parts.Join("-") + "\""
}
