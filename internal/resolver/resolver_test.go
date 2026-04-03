package resolver

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

func TestParseAcceptEncodingPriority(t *testing.T) {
	got := parseAcceptEncoding("gzip;q=0.8, br;q=1.0")
	if len(got) != 2 || got[0] != "br" || got[1] != "gzip" {
		t.Fatalf("unexpected encodings: %#v", got)
	}
}

func TestParseAcceptEncodingWildcard(t *testing.T) {
	got := parseAcceptEncoding("gzip;q=0, *;q=0.5")
	if len(got) != 1 || got[0] != "br" {
		t.Fatalf("unexpected encodings: %#v", got)
	}
}

func TestParseAcceptImageFormatsPriority(t *testing.T) {
	got := parseAcceptImageFormats("image/png;q=1,image/jpeg;q=0.6,*/*;q=0.1", "jpeg")
	if len(got) != 2 || got[0] != "png" || got[1] != "jpeg" {
		t.Fatalf("unexpected image formats: %#v", got)
	}
}

func TestResolverSelectsCompressedVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "index.html")
	variantPath := filepath.Join(dir, "index.html.br")
	if err := os.WriteFile(sourcePath, []byte("<html>origin</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variantPath, []byte("compressed"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "index.html",
		FullPath:   sourcePath,
		MediaType:  "text/html; charset=utf-8",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "index.html.br",
		AssetPath:    "index.html",
		ArtifactPath: variantPath,
		MediaType:    "text/html; charset=utf-8",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-br\"",
		Encoding:     "br",
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config: &config.Assets{
			Entry: "index.html",
			Fallback: config.Fallback{
				On:     config.FallbackOnNotFound,
				Target: "index.html",
			},
		},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{
		Path:           "index.html",
		AcceptEncoding: "br,gzip",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContentEncoding != "br" {
		t.Fatalf("expected br encoding, got %q", result.ContentEncoding)
	}
	if result.ETag != "\"hash-1-br\"" {
		t.Fatalf("expected variant etag, got %q", result.ETag)
	}
}

func TestResolverFallsBackForSPAPath(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(sourcePath, []byte("<html>origin</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "index.html",
		FullPath:   sourcePath,
		MediaType:  "text/html; charset=utf-8",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config: &config.Assets{
			Entry: "index.html",
			Fallback: config.Fallback{
				On:     config.FallbackOnNotFound,
				Target: "index.html",
			},
		},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{Path: "docs"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.FallbackUsed {
		t.Fatal("expected fallback to be used")
	}
	if result.FilePath != sourcePath {
		t.Fatalf("expected fallback path %q, got %q", sourcePath, result.FilePath)
	}
}

func TestResolverSelectsWidthVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.jpg")
	variantPath := filepath.Join(dir, "hero.w640.jpg")
	if err := os.WriteFile(sourcePath, []byte("origin"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variantPath, []byte("scaled"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "hero.jpg",
		FullPath:   sourcePath,
		MediaType:  "image/jpeg",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "hero.jpg|width=640",
		AssetPath:    "hero.jpg",
		ArtifactPath: variantPath,
		MediaType:    "image/jpeg",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-w640\"",
		Width:        640,
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config:  &config.Assets{Entry: "index.html"},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{Path: "hero.jpg", Width: 320})
	if err != nil {
		t.Fatal(err)
	}
	if result.Variant == nil || result.Variant.Width != 640 {
		t.Fatalf("expected 640 width variant, got %#v", result.Variant)
	}
}

func TestResolverRequestsWidthGenerationWhenMissing(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.jpg")
	if err := os.WriteFile(sourcePath, []byte("origin"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "hero.jpg",
		FullPath:   sourcePath,
		MediaType:  "image/jpeg",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config:  &config.Assets{Entry: "index.html"},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{Path: "hero.jpg", Width: 640})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PreferredWidths) != 1 || result.PreferredWidths[0] != 640 {
		t.Fatalf("expected width generation request, got %#v", result.PreferredWidths)
	}
}

func TestResolverSelectsFormatVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.png")
	variantPath := filepath.Join(dir, "hero.fjpeg.jpg")
	if err := os.WriteFile(sourcePath, []byte("origin"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variantPath, []byte("converted"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "hero.png",
		FullPath:   sourcePath,
		MediaType:  "image/png",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "hero.png|format=jpeg",
		AssetPath:    "hero.png",
		ArtifactPath: variantPath,
		MediaType:    "image/jpeg",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-jpeg\"",
		Format:       "jpeg",
		Width:        0,
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config:  &config.Assets{Entry: "index.html"},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{Path: "hero.png", Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Variant == nil || result.Variant.Format != "jpeg" {
		t.Fatalf("expected jpeg format variant, got %#v", result.Variant)
	}
}

func TestResolverRequestsFormatGenerationWhenMissing(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.png")
	if err := os.WriteFile(sourcePath, []byte("origin"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "hero.png",
		FullPath:   sourcePath,
		MediaType:  "image/png",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config:  &config.Assets{Entry: "index.html"},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{Path: "hero.png", Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PreferredFormats) != 1 || result.PreferredFormats[0] != "jpeg" {
		t.Fatalf("expected format generation request, got %#v", result.PreferredFormats)
	}
}

func TestResolverSelectsFormatVariantFromAccept(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.png")
	variantPath := filepath.Join(dir, "hero.fjpeg.jpg")
	if err := os.WriteFile(sourcePath, []byte("origin"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variantPath, []byte("converted"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "hero.png",
		FullPath:   sourcePath,
		MediaType:  "image/png",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cat.UpsertVariant(&catalog.Variant{
		ID:           "hero.png|format=jpeg",
		AssetPath:    "hero.png",
		ArtifactPath: variantPath,
		MediaType:    "image/jpeg",
		SourceHash:   "hash-1",
		ETag:         "\"hash-1-jpeg\"",
		Format:       "jpeg",
		Width:        0,
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config:  &config.Assets{Entry: "index.html"},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{
		Path:   "hero.png",
		Accept: "image/jpeg,image/png;q=0.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Variant == nil || result.Variant.Format != "jpeg" {
		t.Fatalf("expected jpeg format variant, got %#v", result.Variant)
	}
}

func TestResolverRequestsFormatGenerationFromAcceptWhenMissing(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "hero.png")
	if err := os.WriteFile(sourcePath, []byte("origin"), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := catalog.NewInMemoryCatalog()
	if err := cat.UpsertAsset(&catalog.Asset{
		Path:       "hero.png",
		FullPath:   sourcePath,
		MediaType:  "image/png",
		SourceHash: "hash-1",
		ETag:       "\"hash-1\"",
	}); err != nil {
		t.Fatal(err)
	}

	resolver := newResolver(resolverIn{
		Config:  &config.Assets{Entry: "index.html"},
		Catalog: cat,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	result, err := resolver.Resolve(Request{
		Path:   "hero.png",
		Accept: "image/jpeg,image/png;q=0.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PreferredFormats) == 0 || result.PreferredFormats[0] != "jpeg" {
		t.Fatalf("expected format generation request from accept, got %#v", result.PreferredFormats)
	}
}
