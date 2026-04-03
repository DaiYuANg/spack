package pipeline

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/andybalholm/brotli"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
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

func newCompressionStage(in compressionStageIn) Stage {
	return &compressionStage{
		cfg:     in.Config,
		store:   in.Store,
		catalog: in.Catalog,
	}
}

func newCompressionStageFromDeps(cfg *config.Compression, store artifact.Store, cat catalog.Catalog) *compressionStage {
	return newCompressionStage(compressionStageIn{
		Config:  cfg,
		Store:   store,
		Catalog: cat,
	}).(*compressionStage)
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
	tasks := make([]Task, 0, encodings.Len())
	encodings.Range(func(_ int, encoding string) bool {
		if hasEncodingVariant(existing, asset.SourceHash, encoding) {
			return true
		}

		tasks = append(tasks, Task{
			AssetPath: asset.Path,
			Encoding:  encoding,
		})
		return true
	})

	return tasks
}

func (s *compressionStage) Execute(task Task, asset *catalog.Asset) (*catalog.Variant, error) {
	if asset.SourceHash == "" {
		return nil, fmt.Errorf("asset %s missing source hash", asset.Path)
	}

	raw, err := os.ReadFile(asset.FullPath)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) < s.cfg.MinSize {
		return nil, nil
	}

	compressed, suffix, err := s.compress(raw, task.Encoding)
	if err != nil {
		return nil, err
	}
	if len(compressed) >= len(raw) {
		return nil, nil
	}

	targetPath := s.store.PathFor(asset.Path, asset.SourceHash, "encoding", suffix)
	if err := s.store.Write(targetPath, compressed); err != nil {
		return nil, err
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
			"stage": "compression",
		},
	}, nil
}

func (s *compressionStage) compress(raw []byte, encoding string) ([]byte, string, error) {
	switch encoding {
	case "br":
		var buf bytes.Buffer
		writer := brotli.NewWriterLevel(&buf, clampBrotliQuality(s.cfg.BrotliQuality))
		if _, err := writer.Write(raw); err != nil {
			return nil, "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), ".br", nil
	case "gzip":
		var buf bytes.Buffer
		writer, err := gzip.NewWriterLevel(&buf, clampGzipLevel(s.cfg.GzipLevel))
		if err != nil {
			return nil, "", err
		}
		if _, err := writer.Write(raw); err != nil {
			return nil, "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), ".gz", nil
	default:
		return nil, "", fmt.Errorf("unsupported compression encoding: %s", encoding)
	}
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
	if mime == "" {
		return false
	}
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	if strings.HasPrefix(mime, "image/") && mime != "image/svg+xml" {
		return false
	}
	if strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") {
		return false
	}
	if strings.Contains(mime, "zip") || strings.Contains(mime, "gzip") {
		return false
	}
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
