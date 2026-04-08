package sourcecatalog_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/internal/sourcecatalog"
)

func TestScannerRecognizesEnabledCompressionSidecars(t *testing.T) {
	root := t.TempDir()
	writeSourceFile(t, filepath.Join(root, "app.js"), []byte("console.log('ok');"))
	sidecarPath := filepath.Join(root, "app.js.br")
	writeSourceFile(t, sidecarPath, []byte("compressed"))

	scanner := newScannerForTest(t, root, config.DefaultConfigForTest().Compression.NormalizedEncodings())
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := snapshot.Assets.Get("app.js"); !ok {
		t.Fatal("expected app.js asset to be scanned")
	}
	if _, ok := snapshot.Assets.Get("app.js.br"); ok {
		t.Fatal("expected app.js.br to be recognized as sidecar variant, not asset")
	}

	variant, ok := snapshot.Variants.Get("app.js.br")
	if !ok {
		t.Fatal("expected app.js.br variant to be registered")
	}
	if variant.ArtifactPath != sidecarPath {
		t.Fatalf("expected sidecar artifact path %q, got %q", sidecarPath, variant.ArtifactPath)
	}
	if variant.Encoding != "br" {
		t.Fatalf("expected br encoding, got %q", variant.Encoding)
	}
	if !sourcecatalog.IsSourceSidecarVariant(variant) {
		t.Fatal("expected sidecar variant metadata to mark source sidecar stage")
	}
}

func TestScannerLeavesDisabledEncodingSidecarsAsAssets(t *testing.T) {
	root := t.TempDir()
	writeSourceFile(t, filepath.Join(root, "app.js"), []byte("console.log('ok');"))
	writeSourceFile(t, filepath.Join(root, "app.js.br"), []byte("compressed"))

	scanner := newScannerForTest(t, root, collectionx.NewList("gzip"))
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := snapshot.Assets.Get("app.js.br"); !ok {
		t.Fatal("expected disabled sidecar suffix to remain a plain asset")
	}
	if snapshot.Variants.Len() != 0 {
		t.Fatalf("expected no recognized variants, got %d", snapshot.Variants.Len())
	}
}

func newScannerForTest(t *testing.T, root string, encodings collectionx.List[string]) sourcecatalog.Scanner {
	t.Helper()

	src, err := source.NewLocalFSForTest(&config.Assets{Root: root}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfigForTest()
	return sourcecatalog.NewScanner(src, contentcoding.NewRegistry(contentcoding.Options{
		BrotliQuality: cfg.Compression.BrotliQuality,
		GzipLevel:     cfg.Compression.GzipLevel,
		ZstdLevel:     cfg.Compression.ZstdLevel,
	}, encodings))
}

func writeSourceFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}
