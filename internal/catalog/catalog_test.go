package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteVariantByArtifactPath(t *testing.T) {
	cat := NewInMemoryCatalog()

	asset := &Asset{
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
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cat.UpsertVariant(&Variant{
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
	if len(cat.ListVariants("app.js")) != 0 {
		t.Fatalf("expected no variants, got %#v", cat.ListVariants("app.js"))
	}
}
