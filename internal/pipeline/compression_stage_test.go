package pipeline_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/pipeline"
)

func TestCompressionStagePlanSkipsExistingVariant(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	variantPath := filepath.Join(t.TempDir(), "app.js.br")
	if err := os.WriteFile(variantPath, []byte("compressed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js.br",
		AssetPath:    asset.Path,
		ArtifactPath: variantPath,
		MediaType:    asset.MediaType,
		SourceHash:   asset.SourceHash,
		Encoding:     "br",
		ETag:         "\"hash-1-br\"",
	}); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewCompressionStageForTest(&config.Compression{
		Enable: true,
		Mode:   config.CompressionModeLazy,
	}, newTestStore(t.TempDir()), cat)

	tasks := stage.Plan(asset, pipeline.Request{
		AssetPath:          asset.Path,
		PreferredEncodings: collectionx.NewList("br", "gzip", "zstd"),
	})
	if tasks.Len() != 2 {
		t.Fatalf("expected two tasks, got %d", tasks.Len())
	}
	values := tasks.Values()
	got := []string{values[0].Encoding, values[1].Encoding}
	if !slices.Equal(got, []string{"gzip", "zstd"}) {
		t.Fatalf("expected gzip and zstd tasks, got %#v", got)
	}
}

func TestCompressionStageExecuteCreatesVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "payload.json")
	raw := []byte(`{"message":"` + strings.Repeat("compressible-payload-", 256) + `"}`)
	if err := os.WriteFile(sourcePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "payload.json",
		FullPath:   sourcePath,
		MediaType:  "application/json",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	store := newTestStore(filepath.Join(dir, "cache"))
	stage := pipeline.NewCompressionStageForTest(&config.Compression{
		Enable:    true,
		Mode:      config.CompressionModeLazy,
		CacheDir:  filepath.Join(dir, "cache"),
		MinSize:   1,
		GzipLevel: 5,
	}, store, cat)

	variant, err := stage.Execute(pipeline.Task{
		AssetPath: asset.Path,
		Encoding:  "gzip",
	}, asset)
	if err != nil {
		t.Fatal(err)
	}
	if variant == nil {
		t.Fatal("expected variant to be created")
	}
	if variant.Encoding != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", variant.Encoding)
	}
	expectedPath := store.PathFor(asset.Path, asset.SourceHash, "encoding", ".gz")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected artifact to exist: %v", err)
	}
}

func TestCompressionStageExecuteCreatesZstdVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "payload.json")
	raw := []byte(`{"message":"` + strings.Repeat("zstd-payload-", 512) + `"}`)
	if err := os.WriteFile(sourcePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "payload.json",
		FullPath:   sourcePath,
		MediaType:  "application/json",
		SourceHash: "hash-zstd",
		ETag:       "\"hash-zstd\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	store := newTestStore(filepath.Join(dir, "cache"))
	stage := pipeline.NewCompressionStageForTest(&config.Compression{
		Enable:    true,
		Mode:      config.CompressionModeLazy,
		CacheDir:  filepath.Join(dir, "cache"),
		MinSize:   1,
		ZstdLevel: 3,
	}, store, cat)

	variant, err := stage.Execute(pipeline.Task{
		AssetPath: asset.Path,
		Encoding:  "zstd",
	}, asset)
	if err != nil {
		t.Fatal(err)
	}
	if variant == nil {
		t.Fatal("expected zstd variant to be created")
	}
	if variant.Encoding != "zstd" {
		t.Fatalf("expected zstd encoding, got %q", variant.Encoding)
	}
	expectedPath := store.PathFor(asset.Path, asset.SourceHash, "encoding", ".zst")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected zstd artifact to exist: %v", err)
	}
}

func TestCompressionStagePlanUsesConfiguredEncodingOrder(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewCompressionStageForTest(&config.Compression{
		Enable:    true,
		Mode:      config.CompressionModeLazy,
		Encodings: "gzip,zstd",
	}, newTestStore(t.TempDir()), cat)

	tasks := stage.Plan(asset, pipeline.Request{AssetPath: asset.Path})
	if tasks.Len() != 2 {
		t.Fatalf("expected two tasks, got %d", tasks.Len())
	}
	values := tasks.Values()
	got := []string{values[0].Encoding, values[1].Encoding}
	if !slices.Equal(got, []string{"gzip", "zstd"}) {
		t.Fatalf("expected configured encoding order [gzip zstd], got %#v", got)
	}
}

func TestCompressionStagePlanFiltersDisabledRequestEncodings(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewCompressionStageForTest(&config.Compression{
		Enable:    true,
		Mode:      config.CompressionModeLazy,
		Encodings: "br,gzip",
	}, newTestStore(t.TempDir()), cat)

	tasks := stage.Plan(asset, pipeline.Request{
		AssetPath:          asset.Path,
		PreferredEncodings: collectionx.NewList("zstd", "gzip"),
	})
	values := tasks.Values()
	if tasks.Len() != 1 || values[0].Encoding != "gzip" {
		t.Fatalf("expected only enabled gzip task, got %#v", values)
	}
}

func TestImageStagePlanSchedulesWidthVariant(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "hero.jpg",
		FullPath:   "hero.jpg",
		MediaType:  "image/jpeg",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewImageStageForTest(&config.Image{
		Enable: true,
		Widths: "640,1280",
	}, newTestStore(t.TempDir()), cat)

	tasks := stage.Plan(asset, pipeline.Request{
		AssetPath:       asset.Path,
		PreferredWidths: collectionx.NewList(640),
	})
	values := tasks.Values()
	if tasks.Len() != 1 || values[0].Width != 640 {
		t.Fatalf("unexpected image tasks: %#v", values)
	}
}

func TestImageStagePlanSchedulesFormatVariant(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "hero.png",
		FullPath:   "hero.png",
		MediaType:  "image/png",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewImageStageForTest(&config.Image{
		Enable: true,
		Widths: "640,1280",
	}, newTestStore(t.TempDir()), cat)

	tasks := stage.Plan(asset, pipeline.Request{
		AssetPath:        asset.Path,
		PreferredFormats: collectionx.NewList("jpeg"),
	})
	if tasks.Len() != 1 {
		t.Fatalf("expected one format task, got %d", tasks.Len())
	}
	values := tasks.Values()
	if values[0].Width != 0 || values[0].Format != "jpeg" {
		t.Fatalf("unexpected format task: %#v", values[0])
	}
}

func TestImageStageExecuteCreatesResizedVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.jpg")
	writeJPEGFixture(t, sourcePath, 1200, 800)

	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "hero.jpg",
		FullPath:   sourcePath,
		Size:       fileSize(t, sourcePath),
		MediaType:  "image/jpeg",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewImageStageForTest(&config.Image{
		Enable:      true,
		JPEGQuality: 70,
	}, newTestStore(filepath.Join(dir, "cache")), cat)

	variant, err := stage.Execute(pipeline.Task{
		AssetPath: asset.Path,
		Width:     640,
	}, asset)
	if err != nil {
		t.Fatal(err)
	}
	if variant == nil {
		t.Fatal("expected image variant to be created")
	}
	if variant.Width != 640 {
		t.Fatalf("expected width 640, got %d", variant.Width)
	}
	if _, err := os.Stat(variant.ArtifactPath); err != nil {
		t.Fatalf("expected artifact to exist: %v", err)
	}
}

func TestImageStageExecuteCreatesFormatVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.png")
	writePNGFixture(t, sourcePath, 1200, 800)

	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "hero.png",
		FullPath:   sourcePath,
		Size:       fileSize(t, sourcePath),
		MediaType:  "image/png",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	stage := pipeline.NewImageStageForTest(&config.Image{
		Enable:      true,
		JPEGQuality: 70,
	}, newTestStore(filepath.Join(dir, "cache")), cat)

	variant, err := stage.Execute(pipeline.Task{
		AssetPath: asset.Path,
		Format:    "jpeg",
		Width:     0,
	}, asset)
	if err != nil {
		t.Fatal(err)
	}
	if variant == nil {
		t.Fatal("expected format variant to be created")
	}
	if variant.Format != "jpeg" || variant.MediaType != "image/jpeg" {
		t.Fatalf("unexpected format variant: %#v", variant)
	}
	if _, err := os.Stat(variant.ArtifactPath); err != nil {
		t.Fatalf("expected artifact to exist: %v", err)
	}
}
