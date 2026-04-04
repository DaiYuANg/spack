package config_test

import (
	"testing"
	"time"

	"github.com/daiyuang/spack/internal/config"
)

func TestMemoryCacheParsedTTL(t *testing.T) {
	cfg := config.MemoryCache{TTL: "2m"}
	if got := cfg.ParsedTTL(); got != 2*time.Minute {
		t.Fatalf("expected 2m TTL, got %s", got)
	}

	cfg.TTL = "bad"
	if got := cfg.ParsedTTL(); got != 5*time.Minute {
		t.Fatalf("expected fallback TTL 5m, got %s", got)
	}
}

func TestMemoryCacheEnabled(t *testing.T) {
	cfg := config.MemoryCache{
		Enable:      true,
		Warmup:      true,
		MaxEntries:  128,
		MaxFileSize: 64 * 1024,
		TTL:         "5m",
	}
	if !cfg.Enabled() {
		t.Fatal("expected memory cache to be enabled")
	}

	cfg.MaxEntries = 0
	if cfg.Enabled() {
		t.Fatal("expected memory cache to be disabled when max entries is zero")
	}
}

func TestMemoryCacheWarmupEnabled(t *testing.T) {
	cfg := config.MemoryCache{
		Enable:      true,
		Warmup:      true,
		MaxEntries:  128,
		MaxFileSize: 64 * 1024,
		TTL:         "5m",
	}
	if !cfg.WarmupEnabled() {
		t.Fatal("expected memory cache warmup to be enabled")
	}

	cfg.Warmup = false
	if cfg.WarmupEnabled() {
		t.Fatal("expected memory cache warmup to be disabled")
	}
}
