package task_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/internal/task"
)

func TestSyncSourceCatalogRemovesDeletedAssetsAndVariants(t *testing.T) {
	root := t.TempDir()
	src := newLocalSourceForTest(t, root)
	cat := catalog.NewInMemoryCatalog()
	artifactPath := filepath.Join(root, "cache", "app.js.br")

	upsertAssetForTest(t, cat, &catalog.Asset{
		Path:       "app.js",
		FullPath:   filepath.Join(root, "app.js"),
		MediaType:  "application/javascript",
		SourceHash: "hash-old",
		ETag:       "\"hash-old\"",
	})
	upsertVariantForTest(t, cat, artifactPath)

	report, err := task.SyncSourceCatalogForTest(context.Background(), src, cat, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertOne(t, report.Removed, "removed asset")
	assertOne(t, report.RemovedVariants, "removed variant")
	if _, ok := cat.FindAsset("app.js"); ok {
		t.Fatal("expected app.js to be removed from catalog")
	}
	assertFileRemoved(t, artifactPath)
}

func TestSyncSourceCatalogRemovesVariantsForChangedAsset(t *testing.T) {
	root := t.TempDir()
	assetPath := filepath.Join(root, "app.js")
	writeFileForTest(t, assetPath, []byte("console.log('new');"))

	src := newLocalSourceForTest(t, root)
	cat := catalog.NewInMemoryCatalog()
	artifactPath := filepath.Join(root, "cache", "app.js.br")

	upsertAssetForTest(t, cat, &catalog.Asset{
		Path:       "app.js",
		FullPath:   assetPath,
		Size:       3,
		MediaType:  "application/javascript",
		SourceHash: "hash-old",
		ETag:       "\"hash-old\"",
	})
	upsertVariantForTest(t, cat, artifactPath)

	report, err := task.SyncSourceCatalogForTest(context.Background(), src, cat, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertOne(t, report.Updated, "updated asset")
	assertOne(t, report.RemovedVariants, "removed variant")
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
	assertFileRemoved(t, artifactPath)
}

func newLocalSourceForTest(t *testing.T, root string) source.Source {
	t.Helper()

	src, err := source.NewLocalFSForTest(&config.Assets{Root: root}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func upsertAssetForTest(t *testing.T, cat catalog.Catalog, asset *catalog.Asset) {
	t.Helper()
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}
}

func upsertVariantForTest(t *testing.T, cat catalog.Catalog, artifactPath string) {
	t.Helper()

	writeFileForTest(t, artifactPath, []byte("payload"))
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
}

func writeFileForTest(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertOne(t *testing.T, got int, name string) {
	t.Helper()
	if got != 1 {
		t.Fatalf("expected %s 1, got %d", name, got)
	}
}

func assertFileRemoved(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be deleted, got err=%v", path, err)
	}
}
