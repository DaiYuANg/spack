package pipeline_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/eventx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	appEvent "github.com/daiyuang/spack/internal/event"
	"github.com/daiyuang/spack/internal/pipeline"
)

func TestEnqueueDeduplicatesRequests(t *testing.T) {
	svc := pipeline.NewServiceForTest(&config.Compression{
		Enable: true,
		Mode:   config.CompressionModeLazy,
	}, slog.New(slog.DiscardHandler), catalog.NewInMemoryCatalog(), 2)

	svc.Enqueue(pipeline.Request{
		AssetPath:          "hero.png",
		PreferredEncodings: collectionx.NewList("gzip", "br"),
		PreferredFormats:   collectionx.NewList("jpeg", "png"),
		PreferredWidths:    collectionx.NewList(1280, 640),
	})
	svc.Enqueue(pipeline.Request{
		AssetPath:          "hero.png",
		PreferredEncodings: collectionx.NewList("br", "gzip"),
		PreferredFormats:   collectionx.NewList("png", "jpeg"),
		PreferredWidths:    collectionx.NewList(640, 1280),
	})

	if pipeline.QueuedCountForTest(svc) != 1 {
		t.Fatalf("expected one queued request, got %d", pipeline.QueuedCountForTest(svc))
	}
	if pipeline.PendingCountForTest(svc) != 1 {
		t.Fatalf("expected one pending request, got %d", pipeline.PendingCountForTest(svc))
	}
}

func TestEnqueueDropsWhenQueueFull(t *testing.T) {
	svc := pipeline.NewServiceForTest(&config.Compression{
		Enable: true,
		Mode:   config.CompressionModeLazy,
	}, slog.New(slog.DiscardHandler), catalog.NewInMemoryCatalog(), 1)

	svc.Enqueue(pipeline.Request{AssetPath: "a.js"})
	svc.Enqueue(pipeline.Request{AssetPath: "b.js"})

	if pipeline.QueuedCountForTest(svc) != 1 {
		t.Fatalf("expected one queued request, got %d", pipeline.QueuedCountForTest(svc))
	}
	if pipeline.PendingCountForTest(svc) != 1 {
		t.Fatalf("expected one pending request, got %d", pipeline.PendingCountForTest(svc))
	}
}

func TestCleanupArtifactsRemovesExpiredVariants(t *testing.T) {
	root := t.TempDir()
	cat := catalog.NewInMemoryCatalog()
	addAsset(t, cat, "app.js", "application/javascript", "hash-1")

	expiredPath := filepath.Join(root, "encoding", "hash-1", "app.js.br")
	writeArtifact(t, expiredPath, []byte("expired"))
	setMTime(t, expiredPath, time.Now().Add(-2*time.Hour))

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=br",
		AssetPath:    "app.js",
		ArtifactPath: expiredPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	svc := pipeline.NewServiceForTest(&config.Compression{
		CacheDir: root,
		MaxAge:   "1h",
	}, slog.New(slog.DiscardHandler), cat, 1)

	if removed := pipeline.CleanupRemovedForTest(svc, time.Now()); removed != 1 {
		t.Fatalf("expected one removed file, got %d", removed)
	}
	if _, err := os.Stat(expiredPath); !os.IsNotExist(err) {
		t.Fatalf("expected expired file removed, err=%v", err)
	}
	if cat.ListVariants("app.js").Len() != 0 {
		t.Fatalf("expected variant removed from catalog, got %#v", cat.ListVariants("app.js"))
	}
}

func TestCleanupArtifactsEnforcesMaxCacheBytes(t *testing.T) {
	root := t.TempDir()
	cat := catalog.NewInMemoryCatalog()
	addAsset(t, cat, "bundle.js", "application/javascript", "hash-2")

	oldPath := filepath.Join(root, "encoding", "hash-2", "bundle.js.br")
	newPath := filepath.Join(root, "encoding", "hash-2", "bundle.js.gz")
	writeArtifact(t, oldPath, []byte("0123456789abcdef"))
	writeArtifact(t, newPath, []byte("0123456789abcdef"))
	setMTime(t, oldPath, time.Now().Add(-2*time.Hour))
	setMTime(t, newPath, time.Now().Add(-1*time.Hour))

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "bundle.js|encoding=br",
		AssetPath:    "bundle.js",
		ArtifactPath: oldPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-2",
		ETag:         "\"hash-2-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "bundle.js|encoding=gzip",
		AssetPath:    "bundle.js",
		ArtifactPath: newPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-2",
		ETag:         "\"hash-2-gzip\"",
		Encoding:     "gzip",
	}); err != nil {
		t.Fatal(err)
	}

	svc := pipeline.NewServiceForTest(&config.Compression{
		CacheDir:      root,
		MaxCacheBytes: 16,
	}, slog.New(slog.DiscardHandler), cat, 1)

	if removed := pipeline.CleanupRemovedForTest(svc, time.Now()); removed != 1 {
		t.Fatalf("expected one removed file, got %d", removed)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected oldest file removed, err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected newest file retained, err=%v", err)
	}
	variants := cat.ListVariants("bundle.js")
	first, ok := variants.GetFirst()
	if !ok || variants.Len() != 1 || first.Encoding != "gzip" {
		t.Fatalf("expected only gzip variant retained, got %#v", variants)
	}
}

func TestCleanupArtifactsUsesNamespaceMaxAge(t *testing.T) {
	root := t.TempDir()
	cat := catalog.NewInMemoryCatalog()
	addAsset(t, cat, "hero.png", "image/png", "hash-3")

	oldImage := filepath.Join(root, "image", "hash-3", "hero.png.fjpeg.jpg")
	writeArtifact(t, oldImage, []byte("variant"))
	setMTime(t, oldImage, time.Now().Add(-2*time.Hour))

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "hero.png|format=jpeg",
		AssetPath:    "hero.png",
		ArtifactPath: oldImage,
		MediaType:    "image/jpeg",
		SourceHash:   "hash-3",
		ETag:         "\"hash-3-jpeg\"",
		Format:       "jpeg",
	}); err != nil {
		t.Fatal(err)
	}

	svc := pipeline.NewServiceForTest(&config.Compression{
		CacheDir:    root,
		MaxAge:      "24h",
		ImageMaxAge: "1h",
	}, slog.New(slog.DiscardHandler), cat, 1)

	if removed := pipeline.CleanupRemovedForTest(svc, time.Now()); removed != 1 {
		t.Fatalf("expected one removed file, got %d", removed)
	}
	if _, err := os.Stat(oldImage); !os.IsNotExist(err) {
		t.Fatalf("expected old image file removed, err=%v", err)
	}
}

func TestCleanupArtifactsKeepsHotVariant(t *testing.T) {
	root := t.TempDir()
	cat := catalog.NewInMemoryCatalog()
	addAsset(t, cat, "bundle.js", "application/javascript", "hash-4")

	oldPath := filepath.Join(root, "encoding", "hash-4", "bundle.js.br")
	writeArtifact(t, oldPath, []byte("compressed"))
	setMTime(t, oldPath, time.Now().Add(-3*time.Hour))

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "bundle.js|encoding=br",
		AssetPath:    "bundle.js",
		ArtifactPath: oldPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-4",
		ETag:         "\"hash-4-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	svc := pipeline.NewServiceForTest(&config.Compression{
		CacheDir: root,
		MaxAge:   "1h",
	}, slog.New(slog.DiscardHandler), cat, 1)
	svc.MarkVariantHit(oldPath)

	if removed := pipeline.CleanupRemovedForTest(svc, time.Now()); removed != 0 {
		t.Fatalf("expected no removed files for hot variant, got %d", removed)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected hot variant retained, err=%v", err)
	}
}

func TestVariantServedEventKeepsHotVariant(t *testing.T) {
	root := t.TempDir()
	cat := catalog.NewInMemoryCatalog()
	addAsset(t, cat, "bundle.js", "application/javascript", "hash-5")

	oldPath := filepath.Join(root, "encoding", "hash-5", "bundle.js.br")
	writeArtifact(t, oldPath, []byte("compressed"))
	setMTime(t, oldPath, time.Now().Add(-3*time.Hour))

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "bundle.js|encoding=br",
		AssetPath:    "bundle.js",
		ArtifactPath: oldPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-5",
		ETag:         "\"hash-5-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	bus := eventx.New()
	svc := pipeline.NewServiceWithBusForTest(&config.Compression{
		CacheDir: root,
		MaxAge:   "1h",
	}, slog.New(slog.DiscardHandler), cat, bus, 1)
	if err := pipeline.SubscribeVariantServedForTest(svc); err != nil {
		t.Fatal(err)
	}

	if err := bus.Publish(context.Background(), appEvent.VariantServed{
		AssetPath:    "bundle.js",
		ArtifactPath: oldPath,
		ServedAt:     time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	if removed := pipeline.CleanupRemovedForTest(svc, time.Now()); removed != 0 {
		t.Fatalf("expected no removed files for recently served variant, got %d", removed)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected served variant retained, err=%v", err)
	}
}

func addAsset(t *testing.T, cat catalog.Catalog, path, mediaType, sourceHash string) {
	t.Helper()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       path,
		FullPath:   filepath.Join(t.TempDir(), path),
		MediaType:  mediaType,
		SourceHash: sourceHash,
		ETag:       "\"" + sourceHash + "\"",
	}); err != nil {
		t.Fatal(err)
	}
}

func writeArtifact(t *testing.T, path string, payload []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
}

func setMTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}
