package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DaiYuANg/arcgo/configx"
)

func TestLoadIntoDefaultConfigPreservesNestedDefaultsWithPartialDotenv(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	envBody := "APP_ASSETS_ROOT=/tmp/assets\nAPP_ASSETS_PATH=/\nAPP_COMPRESSION_ENABLE=true\n"
	if err := os.WriteFile(envPath, []byte(envBody), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig()
	if err := configx.Load(
		&cfg,
		configx.WithEnvPrefix("APP"),
		configx.WithIgnoreDotenvError(false),
		configx.WithDotenv(envPath),
	); err != nil {
		t.Fatal(err)
	}

	if cfg.Assets.Entry != "index.html" {
		t.Fatalf("expected default assets entry to be preserved, got %q", cfg.Assets.Entry)
	}
	if cfg.Assets.Fallback.On != FallbackOnNotFound {
		t.Fatalf("expected default fallback mode %q, got %q", FallbackOnNotFound, cfg.Assets.Fallback.On)
	}
	if cfg.Debug.LivePort != 8080 {
		t.Fatalf("expected default debug live port 8080, got %d", cfg.Debug.LivePort)
	}
	if cfg.Compression.CacheDir == "" {
		t.Fatal("expected default compression cache dir to be preserved")
	}
	if cfg.Compression.Workers != 2 {
		t.Fatalf("expected default compression workers 2, got %d", cfg.Compression.Workers)
	}
}
