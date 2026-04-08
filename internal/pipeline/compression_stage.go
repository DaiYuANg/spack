// Package pipeline plans and executes derived asset generation.
package pipeline

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/samber/lo"
)

type compressionStage struct {
	cfg        *config.Compression
	store      artifact.Store
	catalog    catalog.Catalog
	strategies contentcoding.Registry
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
		strategies: contentcoding.NewRegistry(contentcoding.Options{
			BrotliQuality: in.Config.BrotliQuality,
			GzipLevel:     in.Config.GzipLevel,
			ZstdLevel:     in.Config.ZstdLevel,
		}, in.Config.NormalizedEncodings()),
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

	supportedEncodings := s.cfg.NormalizedEncodings()
	encodings := filterConfiguredEncodings(normalizeEncodings(request.PreferredEncodings), supportedEncodings)
	if encodings.IsEmpty() {
		encodings = supportedEncodings
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
		Metadata: collectionx.NewMapFrom(map[string]string{
			"stage":      "compression",
			"mtime_unix": strconv.FormatInt(time.Now().Unix(), 10),
		}),
	}, nil
}

func (s *compressionStage) compress(raw []byte, encoding string) ([]byte, string, error) {
	strategy, ok := s.strategies.Lookup(encoding)
	if !ok {
		return nil, "", fmt.Errorf("unsupported compression encoding: %s", encoding)
	}

	compressed, err := strategy.Compress(raw)
	if err != nil {
		return nil, "", err
	}
	return compressed, strategy.Suffix(), nil
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
	return contentcoding.NormalizeNames(encodings)
}

func filterConfiguredEncodings(encodings, supported collectionx.List[string]) collectionx.List[string] {
	return collectionx.FilterMapList(encodings, func(_ int, encoding string) (string, bool) {
		return encoding, lo.Contains(supported.Values(), encoding)
	})
}
