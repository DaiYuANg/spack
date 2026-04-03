package pipeline

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
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
	if err := os.WriteFile(variantPath, []byte("compressed"), 0o644); err != nil {
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

	stage := newCompressionStage(compressionStageIn{
		Config: &config.Compression{
			Enable: true,
			Mode:   config.CompressionModeLazy,
		},
		Store:   newTestStore(t.TempDir()),
		Catalog: cat,
	}).(*compressionStage)

	tasks := stage.Plan(asset, Request{
		AssetPath:          asset.Path,
		PreferredEncodings: []string{"br", "gzip"},
	})
	if len(tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(tasks))
	}
	if tasks[0].Encoding != "gzip" {
		t.Fatalf("expected gzip task, got %q", tasks[0].Encoding)
	}
}

func TestCompressionStageExecuteCreatesVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "payload.json")
	raw := []byte(`{"message":"` + strings.Repeat("compressible-payload-", 256) + `"}`)
	if err := os.WriteFile(sourcePath, raw, 0o644); err != nil {
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

	stage := newCompressionStage(compressionStageIn{
		Config: &config.Compression{
			Enable:    true,
			Mode:      config.CompressionModeLazy,
			CacheDir:  filepath.Join(dir, "cache"),
			MinSize:   1,
			GzipLevel: 5,
		},
		Store:   newTestStore(filepath.Join(dir, "cache")),
		Catalog: cat,
	}).(*compressionStage)

	variant, err := stage.Execute(Task{
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
	if _, err := os.Stat(variant.ArtifactPath); err != nil {
		t.Fatalf("expected artifact to exist: %v", err)
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

	stage := newImageStage(imageStageIn{
		Config: &config.Image{
			Enable: true,
			Widths: "640,1280",
		},
		Store:   newTestStore(t.TempDir()),
		Catalog: cat,
	}).(*imageStage)

	tasks := stage.Plan(asset, Request{AssetPath: asset.Path, PreferredWidths: []int{640}})
	if len(tasks) != 1 || tasks[0].Width != 640 {
		t.Fatalf("unexpected image tasks: %#v", tasks)
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

	stage := newImageStage(imageStageIn{
		Config: &config.Image{
			Enable: true,
			Widths: "640,1280",
		},
		Store:   newTestStore(t.TempDir()),
		Catalog: cat,
	}).(*imageStage)

	tasks := stage.Plan(asset, Request{
		AssetPath:        asset.Path,
		PreferredFormats: []string{"jpeg"},
	})
	if len(tasks) != 1 {
		t.Fatalf("expected one format task, got %d", len(tasks))
	}
	if tasks[0].Width != 0 || tasks[0].Format != "jpeg" {
		t.Fatalf("unexpected format task: %#v", tasks[0])
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

	stage := newImageStage(imageStageIn{
		Config: &config.Image{
			Enable:      true,
			JPEGQuality: 70,
		},
		Store:   newTestStore(filepath.Join(dir, "cache")),
		Catalog: cat,
	}).(*imageStage)

	variant, err := stage.Execute(Task{
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

	stage := newImageStage(imageStageIn{
		Config: &config.Image{
			Enable:      true,
			JPEGQuality: 70,
		},
		Store:   newTestStore(filepath.Join(dir, "cache")),
		Catalog: cat,
	}).(*imageStage)

	variant, err := stage.Execute(Task{
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

func newTestStore(root string) artifact.Store {
	return &testStore{root: root}
}

type testStore struct {
	root string
}

func (s *testStore) Root() string {
	return s.root
}

func (s *testStore) PathFor(assetPath, sourceHash, namespace, suffix string) string {
	cleanPath := filepath.Clean(filepath.FromSlash(assetPath))
	return filepath.Join(s.root, namespace, sourceHash, cleanPath+suffix)
}

func (s *testStore) Write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmpPath := path + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func writeJPEGFixture(t *testing.T, path string, width int, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 255),
				G: uint8(y % 255),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 92}); err != nil {
		t.Fatal(err)
	}
}

func writePNGFixture(t *testing.T, path string, width int, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 255),
				G: uint8(y % 255),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}
