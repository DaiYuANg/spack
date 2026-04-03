package pipeline

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

func TestEnqueueDeduplicatesRequests(t *testing.T) {
	svc := &Service{
		cfg: &config.Compression{
			Enable: true,
			Mode:   config.CompressionModeLazy,
		},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		tasks:   make(chan Request, 2),
		pending: collectionset.NewSet[string](),
	}

	svc.Enqueue(Request{
		AssetPath:          "hero.png",
		PreferredEncodings: []string{"gzip", "br"},
		PreferredFormats:   []string{"jpeg", "png"},
		PreferredWidths:    []int{1280, 640},
	})
	svc.Enqueue(Request{
		AssetPath:          "hero.png",
		PreferredEncodings: []string{"br", "gzip"},
		PreferredFormats:   []string{"png", "jpeg"},
		PreferredWidths:    []int{640, 1280},
	})

	if len(svc.tasks) != 1 {
		t.Fatalf("expected one queued request, got %d", len(svc.tasks))
	}
	if pendingCount(svc) != 1 {
		t.Fatalf("expected one pending request, got %d", pendingCount(svc))
	}
}

func TestEnqueueDropsWhenQueueFull(t *testing.T) {
	svc := &Service{
		cfg: &config.Compression{
			Enable: true,
			Mode:   config.CompressionModeLazy,
		},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		tasks:   make(chan Request, 1),
		pending: collectionset.NewSet[string](),
	}

	svc.Enqueue(Request{AssetPath: "a.js"})
	svc.Enqueue(Request{AssetPath: "b.js"})

	if len(svc.tasks) != 1 {
		t.Fatalf("expected one queued request, got %d", len(svc.tasks))
	}
	if pendingCount(svc) != 1 {
		t.Fatalf("expected one pending request, got %d", pendingCount(svc))
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

	svc := &Service{
		cfg: &config.Compression{
			CacheDir: root,
		},
		logger:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		catalog:                cat,
		cleanupDefaultMaxAge:   time.Hour,
		cleanupMaxCacheBytes:   0,
		cleanupNamespaceMaxAge: collectionmapping.NewMap[string, time.Duration](),
		variantHits:            collectionmapping.NewMap[string, time.Time](),
	}

	result := svc.cleanupArtifacts(time.Now())
	if result.removed != 1 {
		t.Fatalf("expected one removed file, got %d", result.removed)
	}
	if _, err := os.Stat(expiredPath); !os.IsNotExist(err) {
		t.Fatalf("expected expired file removed, err=%v", err)
	}
	if len(cat.ListVariants("app.js")) != 0 {
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

	svc := &Service{
		cfg: &config.Compression{
			CacheDir: root,
		},
		logger:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		catalog:                cat,
		cleanupDefaultMaxAge:   0,
		cleanupMaxCacheBytes:   16,
		cleanupNamespaceMaxAge: collectionmapping.NewMap[string, time.Duration](),
		variantHits:            collectionmapping.NewMap[string, time.Time](),
	}

	result := svc.cleanupArtifacts(time.Now())
	if result.removed != 1 {
		t.Fatalf("expected one removed file, got %d", result.removed)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected oldest file removed, err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected newest file retained, err=%v", err)
	}
	variants := cat.ListVariants("bundle.js")
	if len(variants) != 1 || variants[0].Encoding != "gzip" {
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

	svc := &Service{
		cfg: &config.Compression{
			CacheDir: root,
		},
		logger:               slog.New(slog.NewTextHandler(io.Discard, nil)),
		catalog:              cat,
		cleanupDefaultMaxAge: 24 * time.Hour,
		cleanupNamespaceMaxAge: collectionmapping.NewMapFrom(map[string]time.Duration{
			"image": time.Hour,
		}),
		variantHits: collectionmapping.NewMap[string, time.Time](),
	}

	result := svc.cleanupArtifacts(time.Now())
	if result.removed != 1 {
		t.Fatalf("expected one removed file, got %d", result.removed)
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

	svc := &Service{
		cfg: &config.Compression{
			CacheDir: root,
		},
		logger:               slog.New(slog.NewTextHandler(io.Discard, nil)),
		catalog:              cat,
		cleanupDefaultMaxAge: time.Hour,
		variantHits: collectionmapping.NewMapFrom(map[string]time.Time{
			oldPath: time.Now(),
		}),
		cleanupNamespaceMaxAge: collectionmapping.NewMap[string, time.Duration](),
	}

	result := svc.cleanupArtifacts(time.Now())
	if result.removed != 0 {
		t.Fatalf("expected no removed files for hot variant, got %d", result.removed)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected hot variant retained, err=%v", err)
	}
}

func pendingCount(svc *Service) int {
	svc.pendingMu.Lock()
	defer svc.pendingMu.Unlock()
	return svc.pending.Len()
}

func addAsset(t *testing.T, cat catalog.Catalog, path string, mediaType string, sourceHash string) {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
}

func setMTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}
