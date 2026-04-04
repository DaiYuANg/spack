// Package pipeline plans and executes derived asset generation.
package pipeline

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/andybalholm/brotli"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/samber/lo"
)

type compressionStage struct {
	cfg     *config.Compression
	store   artifact.Store
	catalog catalog.Catalog
}

type compressionStageIn struct {
	Config  *config.Compression
	Store   artifact.Store
	Catalog catalog.Catalog
}

func newCompressionStage(in compressionStageIn) *compressionStage {
	return &compressionStage{
		cfg:     in.Config,
		store:   in.Store,
		catalog: in.Catalog,
	}
}

func newCompressionStageFromDeps(cfg *config.Compression, store artifact.Store, cat catalog.Catalog) *compressionStage {
	return newCompressionStage(compressionStageIn{Config: cfg, Store: store, Catalog: cat})
}

func (s *compressionStage) Name() string {
	return "compression"
}

func (s *compressionStage) Plan(asset *catalog.Asset, request Request) []Task {
	if !s.cfg.PipelineEnabled() || !isCompressible(asset) {
		return nil
	}

	encodings := normalizeEncodings(request.PreferredEncodings)
	if encodings.IsEmpty() {
		encodings = collectionx.NewList("br", "gzip")
	}

	existing := s.catalog.ListVariants(asset.Path)
	return lo.FilterMap(encodings.Values(), func(encoding string, _ int) (Task, bool) {
		if hasEncodingVariant(existing, asset.SourceHash, encoding) {
			return Task{}, false
		}
		return Task{
			AssetPath: asset.Path,
			Encoding:  encoding,
		}, true
	})
}

func (s *compressionStage) Execute(task Task, asset *catalog.Asset) (*catalog.Variant, error) {
	if asset.SourceHash == "" {
		return nil, fmt.Errorf("asset %s missing source hash", asset.Path)
	}

	raw, err := os.ReadFile(asset.FullPath)
	if err != nil {
		return nil, fmt.Errorf("read asset payload: %w", err)
	}
	if int64(len(raw)) < s.cfg.MinSize {
		return nil, ErrVariantSkipped
	}

	compressed, suffix, err := s.compress(raw, task.Encoding)
	if err != nil {
		return nil, fmt.Errorf("compress asset payload: %w", err)
	}
	if len(compressed) >= len(raw) {
		return nil, ErrVariantSkipped
	}

	targetPath := s.store.PathFor(asset.Path, asset.SourceHash, "encoding", suffix)
	if err := s.store.Write(targetPath, compressed); err != nil {
		return nil, fmt.Errorf("write compressed artifact: %w", err)
	}

	return &catalog.Variant{
		ID:           asset.Path + suffix,
		AssetPath:    asset.Path,
		ArtifactPath: targetPath,
		Size:         int64(len(compressed)),
		MediaType:    asset.MediaType,
		SourceHash:   asset.SourceHash,
		ETag:         fmt.Sprintf("\"%s-%s\"", asset.SourceHash, task.Encoding),
		Encoding:     task.Encoding,
		Metadata: map[string]string{
			"stage":      "compression",
			"mtime_unix": strconv.FormatInt(time.Now().Unix(), 10),
		},
	}, nil
}

func (s *compressionStage) compress(raw []byte, encoding string) ([]byte, string, error) {
	switch encoding {
	case "br":
		return s.compressBrotli(raw)
	case "gzip":
		return s.compressGzip(raw)
	default:
		return nil, "", fmt.Errorf("unsupported compression encoding: %s", encoding)
	}
}

func (s *compressionStage) compressBrotli(raw []byte) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriterLevel(&buf, clampBrotliQuality(s.cfg.BrotliQuality))
	if _, err := writer.Write(raw); err != nil {
		return nil, "", fmt.Errorf("write brotli payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close brotli writer: %w", err)
	}
	return buf.Bytes(), ".br", nil
}

func (s *compressionStage) compressGzip(raw []byte) ([]byte, string, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, clampGzipLevel(s.cfg.GzipLevel))
	if err != nil {
		return nil, "", fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := writer.Write(raw); err != nil {
		return nil, "", fmt.Errorf("write gzip payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close gzip writer: %w", err)
	}
	return buf.Bytes(), ".gz", nil
}

func hasEncodingVariant(variants collectionx.List[*catalog.Variant], sourceHash, encoding string) bool {
	found := false
	variants.Range(func(_ int, variant *catalog.Variant) bool {
		if variant.Encoding != encoding {
			return true
		}
		if sourceHash != "" && variant.SourceHash != "" && variant.SourceHash != sourceHash {
			return true
		}
		if variant.ArtifactPath == "" {
			return true
		}
		if _, err := os.Stat(variant.ArtifactPath); err != nil {
			return true
		}
		found = true
		return false
	})
	return found
}

func isCompressible(asset *catalog.Asset) bool {
	mime := strings.ToLower(strings.TrimSpace(asset.MediaType))
	switch {
	case mime == "":
		return false
	case strings.HasPrefix(mime, "text/"):
		return true
	case isNonCompressibleMediaType(mime):
		return false
	default:
		return isKnownCompressibleType(mime)
	}
}

func isNonCompressibleMediaType(mime string) bool {
	if strings.HasPrefix(mime, "image/") && mime != "image/svg+xml" {
		return true
	}
	if strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") {
		return true
	}
	return strings.Contains(mime, "zip") || strings.Contains(mime, "gzip")
}

func isKnownCompressibleType(mime string) bool {
	switch mime {
	case "application/javascript",
		"application/json",
		"application/wasm",
		"application/xml",
		"image/svg+xml":
		return true
	default:
		return false
	}
}

func normalizeEncodings(encodings collectionx.List[string]) collectionx.List[string] {
	if encodings.IsEmpty() {
		return collectionx.NewList[string]()
	}

	ordered := collectionx.NewOrderedSetWithCapacity[string](encodings.Len())
	encodings.Each(func(_ int, raw string) {
		encoding := strings.ToLower(strings.TrimSpace(raw))
		switch encoding {
		case "br", "gzip":
			ordered.Add(encoding)
		}
	})
	return collectionx.NewList(ordered.Values()...)
}

func clampGzipLevel(level int) int {
	if level < gzip.BestSpeed || level > gzip.BestCompression {
		return gzip.DefaultCompression
	}
	return level
}

func clampBrotliQuality(q int) int {
	if q < 0 {
		return 0
	}
	if q > 11 {
		return 11
	}
	return q
}
