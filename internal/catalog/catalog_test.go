package catalog_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/daiyuang/spack/internal/catalog"
)

func TestDeleteVariantByArtifactPath(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()

	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "/assets/app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	artifactPath := filepath.Join(root, "encoding", "hash-1", "app.js.br")
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
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	if !cat.DeleteVariantByArtifactPath(artifactPath) {
		t.Fatal("expected variant to be deleted")
	}
	if cat.DeleteVariantByArtifactPath(artifactPath) {
		t.Fatal("expected second delete to return false")
	}
	if cat.ListVariants("app.js").Len() != 0 {
		t.Fatalf("expected no variants, got %#v", cat.ListVariants("app.js"))
	}
}

func TestVariantTableStorageBehavior(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()

	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "/assets/app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=gzip",
		AssetPath:    "app.js",
		ArtifactPath: "/artifacts/app.js.gz",
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-gzip\"",
		Encoding:     "gzip",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=br",
		AssetPath:    "app.js",
		ArtifactPath: "/artifacts/app.js.br",
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	if got := cat.VariantCount(); got != 2 {
		t.Fatalf("expected variant count 2, got %d", got)
	}

	got := cat.ListVariants("app.js")
	first, ok := got.Get(0)
	if !ok {
		t.Fatal("expected first variant")
	}
	second, ok := got.Get(1)
	if !ok {
		t.Fatal("expected second variant")
	}
	gotIDs := []string{first.ID, second.ID}
	wantIDs := []string{"app.js|encoding=br", "app.js|encoding=gzip"}
	if !slices.Equal(gotIDs, wantIDs) {
		t.Fatalf("expected sorted variants %v, got %v", wantIDs, gotIDs)
	}
}

func TestUpsertVariantRefreshesArtifactIndexOnOverwrite(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()

	asset := &catalog.Asset{
		Path:       "bundle.js",
		FullPath:   "/assets/bundle.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "bundle.js|encoding=br",
		AssetPath:    "bundle.js",
		ArtifactPath: "/artifacts/old.br",
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br-old\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "bundle.js|encoding=br",
		AssetPath:    "bundle.js",
		ArtifactPath: "/artifacts/new.br",
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br-new\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	if cat.DeleteVariantByArtifactPath("/artifacts/old.br") {
		t.Fatal("expected stale artifact index to be removed")
	}
	if !cat.DeleteVariantByArtifactPath("/artifacts/new.br") {
		t.Fatal("expected latest artifact path to resolve through index")
	}
	if got := cat.VariantCount(); got != 0 {
		t.Fatalf("expected variant count 0 after delete, got %d", got)
	}
}

func TestUpsertVariantReplacesArtifactPathCollision(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()

	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "/assets/app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}

	collidingPath := "/artifacts/shared.bin"
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=gzip",
		AssetPath:    "app.js",
		ArtifactPath: collidingPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-gzip\"",
		Encoding:     "gzip",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=br",
		AssetPath:    "app.js",
		ArtifactPath: collidingPath,
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	variants := cat.ListVariants("app.js")
	if got := variants.Len(); got != 1 {
		t.Fatalf("expected artifact path collision to keep one variant, got %d", got)
	}
	first, ok := variants.Get(0)
	if !ok {
		t.Fatal("expected variant after collision")
	}
	if first.ID != "app.js|encoding=br" {
		t.Fatalf("expected latest colliding variant to win, got %q", first.ID)
	}
}

func TestDeleteVariantsRemovesAssetVariantsOnly(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()

	for _, asset := range []*catalog.Asset{
		{
			Path:       "app.js",
			FullPath:   "/assets/app.js",
			MediaType:  "application/javascript",
			SourceHash: "hash-1",
			ETag:       "\"hash-1\"",
		},
		{
			Path:       "style.css",
			FullPath:   "/assets/style.css",
			MediaType:  "text/css",
			SourceHash: "hash-2",
			ETag:       "\"hash-2\"",
		},
	} {
		if err := cat.UpsertAsset(asset); err != nil {
			t.Fatal(err)
		}
	}

	for _, variant := range []*catalog.Variant{
		{
			ID:           "app.js|encoding=br",
			AssetPath:    "app.js",
			ArtifactPath: "/artifacts/app.js.br",
			MediaType:    "application/javascript",
			SourceHash:   "hash-1",
			ETag:         "\"hash-1-br\"",
			Encoding:     "br",
		},
		{
			ID:           "style.css|encoding=gzip",
			AssetPath:    "style.css",
			ArtifactPath: "/artifacts/style.css.gz",
			MediaType:    "text/css",
			SourceHash:   "hash-2",
			ETag:         "\"hash-2-gzip\"",
			Encoding:     "gzip",
		},
	} {
		if err := cat.UpsertVariant(variant); err != nil {
			t.Fatal(err)
		}
	}

	removed := cat.DeleteVariants("app.js")
	if removed.Len() != 1 {
		t.Fatalf("expected one removed variant, got %d", removed.Len())
	}
	if cat.ListVariants("app.js").Len() != 0 {
		t.Fatalf("expected app.js variants removed, got %#v", cat.ListVariants("app.js"))
	}
	if cat.ListVariants("style.css").Len() != 1 {
		t.Fatalf("expected style.css variants to remain, got %#v", cat.ListVariants("style.css"))
	}
}

func TestDeleteAssetRemovesAssetAndVariants(t *testing.T) {
	cat := catalog.NewInMemoryCatalog()
	asset := &catalog.Asset{
		Path:       "app.js",
		FullPath:   "/assets/app.js",
		MediaType:  "application/javascript",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}
	if err := cat.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "app.js|encoding=br",
		AssetPath:    "app.js",
		ArtifactPath: "/artifacts/app.js.br",
		MediaType:    "application/javascript",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	removed := cat.DeleteAsset("app.js")
	if removed.Len() != 1 {
		t.Fatalf("expected one removed variant, got %d", removed.Len())
	}
	if _, ok := cat.FindAsset("app.js"); ok {
		t.Fatal("expected asset to be deleted")
	}
	if cat.VariantCount() != 0 {
		t.Fatalf("expected no variants after delete asset, got %d", cat.VariantCount())
	}
}
