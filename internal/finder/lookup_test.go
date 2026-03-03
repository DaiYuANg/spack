package finder

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/model"
	"github.com/daiyuang/spack/internal/registry"
)

func TestParseAcceptEncoding_PriorityAndQ(t *testing.T) {
	f := &Finder{}
	got := f.parseAcceptEncoding("gzip;q=0.8, br;q=1.0")
	if len(got) != 2 || got[0] != "br" || got[1] != "gzip" {
		t.Fatalf("unexpected encodings: %#v", got)
	}
}

func TestParseAcceptEncoding_Wildcard(t *testing.T) {
	f := &Finder{}
	got := f.parseAcceptEncoding("gzip;q=0, *;q=0.5")
	if len(got) != 1 || got[0] != "br" {
		t.Fatalf("unexpected wildcard encodings: %#v", got)
	}
}

func TestLookup_SelectsVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "index.html")
	variantPath := filepath.Join(dir, "index.html.br")
	if err := os.WriteFile(sourcePath, []byte("<html>origin</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variantPath, []byte("compressed"), 0o644); err != nil {
		t.Fatal(err)
	}

	source := &model.ObjectInfo{
		Key:      "index.html",
		FullPath: sourcePath,
		Mimetype: constant.Html,
		Metadata: map[string]string{
			"source_hash": "h1",
			"etag":        "\"h1\"",
		},
	}
	variant := &model.ObjectInfo{
		Key:      "index.html.br",
		FullPath: variantPath,
		Mimetype: constant.Html,
		Metadata: map[string]string{
			"encoding":    "br",
			"source_hash": "h1",
			"etag":        "\"h1-br\"",
		},
	}
	reg := registry.NewInMemoryRegistry()
	if err := reg.Register(source); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(variant); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterChildren(source, variant); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterParents(variant, source); err != nil {
		t.Fatal(err)
	}
	f := &Finder{
		registry: reg,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	result, err := f.Lookup(NewLookupContext("br,gzip", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Encoding != "br" {
		t.Fatalf("expected br encoding, got %q", result.Encoding)
	}
	if result.ETag != "\"h1-br\"" {
		t.Fatalf("expected variant etag, got %q", result.ETag)
	}
}

func TestLookup_SkipsStaleVariant(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "index.html")
	variantPath := filepath.Join(dir, "index.html.br")
	if err := os.WriteFile(sourcePath, []byte("<html>origin</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(variantPath, []byte("old-compressed"), 0o644); err != nil {
		t.Fatal(err)
	}

	source := &model.ObjectInfo{
		Key:      "index.html",
		FullPath: sourcePath,
		Mimetype: constant.Html,
		Metadata: map[string]string{
			"source_hash": "new_hash",
			"etag":        "\"new_hash\"",
		},
	}
	stale := &model.ObjectInfo{
		Key:      "index.html.br",
		FullPath: variantPath,
		Mimetype: constant.Html,
		Metadata: map[string]string{
			"encoding":    "br",
			"source_hash": "old_hash",
			"etag":        "\"old_hash-br\"",
		},
	}
	reg := registry.NewInMemoryRegistry()
	if err := reg.Register(source); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(stale); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterChildren(source, stale); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterParents(stale, source); err != nil {
		t.Fatal(err)
	}
	f := &Finder{
		registry: reg,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	result, err := f.Lookup(NewLookupContext("br,gzip", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Encoding != "" {
		t.Fatalf("expected no encoding for stale variant, got %q", result.Encoding)
	}
	if result.ETag != "\"new_hash\"" {
		t.Fatalf("expected source etag, got %q", result.ETag)
	}
	if len(result.AcceptEncoding) == 0 || result.AcceptEncoding[0] != "br" {
		t.Fatalf("expected preferred encoding list, got %#v", result.AcceptEncoding)
	}
}
