package task

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
)

func TestSyncSourceCatalogRemovesDeletedAssetsAndVariants(t *testing.T) {
	root := t.TempDir()
	src, err := source.NewLocalFSForTest(&config.Assets{Root: root}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "app.js",
		FullPath:   filepath.Join(root, "app.js"),
		MediaType:  "application/javascript",
		SourceHash: "hash-old",
		ETag:       "\"hash-old\"",
	}); err != nil {
		t.Fatal(err)
	}

	artifactPath := filepath.Join(root, "cache", "app.js.br")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=br",
		AssetPath:    "app.js",
		ArtifactPath: artifactPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-old",
		ETag:         "\"hash-old-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	report, err := syncSourceCatalog(context.Background(), src, cat, nil)
	if err != nil {
		t.Fatal(err)
	}

	if report.removed != 1 {
		t.Fatalf("expected one removed asset, got %d", report.removed)
	}
	if report.removedVariants != 1 {
		t.Fatalf("expected one removed variant, got %d", report.removedVariants)
	}
	if _, ok := cat.FindAsset("app.js"); ok {
		t.Fatal("expected app.js to be removed from catalog")
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact to be deleted, got err=%v", err)
	}
}

func TestSyncSourceCatalogRemovesVariantsForChangedAsset(t *testing.T) {
	root := t.TempDir()
	assetPath := filepath.Join(root, "app.js")
	if err := os.WriteFile(assetPath, []byte("console.log('new');"), 0o600); err != nil {
		t.Fatal(err)
	}

	src, err := source.NewLocalFSForTest(&config.Assets{Root: root}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "app.js",
		FullPath:   assetPath,
		Size:       3,
		MediaType:  "application/javascript",
		SourceHash: "hash-old",
		ETag:       "\"hash-old\"",
	}); err != nil {
		t.Fatal(err)
	}

	artifactPath := filepath.Join(root, "cache", "app.js.br")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=br",
		AssetPath:    "app.js",
		ArtifactPath: artifactPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-old",
		ETag:         "\"hash-old-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	report, err := syncSourceCatalog(context.Background(), src, cat, nil)
	if err != nil {
		t.Fatal(err)
	}

	if report.updated != 1 {
		t.Fatalf("expected one updated asset, got %d", report.updated)
	}
	if report.removedVariants != 1 {
		t.Fatalf("expected one removed variant, got %d", report.removedVariants)
	}
	if cat.ListVariants("app.js").Len() != 0 {
		t.Fatalf("expected app.js variants to be removed, got %#v", cat.ListVariants("app.js"))
	}

	asset, ok := cat.FindAsset("app.js")
	if !ok {
		t.Fatal("expected app.js to remain in catalog")
	}
	if asset.SourceHash == "hash-old" {
		t.Fatal("expected app.js source hash to be refreshed")
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale artifact to be deleted, got err=%v", err)
	}
}
