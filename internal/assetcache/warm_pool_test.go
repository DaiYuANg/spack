package assetcache_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/panjf2000/ants/v2"
)

func TestWarmWithSharedPool(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "index.html")
	secondPath := filepath.Join(root, "about.html")
	writeAssetFile(t, firstPath, []byte("<html>home</html>"))
	writeAssetFile(t, secondPath, []byte("<html>about</html>"))

	cat := catalog.NewInMemoryCatalog()
	upsertAsset(t, cat, &catalog.Asset{
		Path:     "index.html",
		FullPath: firstPath,
		Size:     int64(len("<html>home</html>")),
	})
	upsertAsset(t, cat, &catalog.Asset{
		Path:     "about.html",
		FullPath: secondPath,
		Size:     int64(len("<html>about</html>")),
	})

	pool, err := ants.NewPool(2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	obs := &recordingObservability{}
	cache := assetcache.NewCacheWithPoolForTest(config.MemoryCache{
		Enable:      true,
		Warmup:      true,
		MaxEntries:  16,
		MaxFileSize: 64 * 1024,
		TTL:         "5m",
	}, slog.New(slog.DiscardHandler), obs, pool)

	stats, err := cache.Warm(context.Background(), cat)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entries != 2 {
		t.Fatalf("expected two warmed entries, got %d", stats.Entries)
	}

	assertCacheHitState(t, cache, firstPath, true)
	assertCacheHitState(t, cache, secondPath, true)
	assertCounterValue(t, obs, "asset_cache_warm_entries_total", 2)
	assertCounterValue(t, obs, "asset_cache_warm_bytes_total", int64(len("<html>home</html>")+len("<html>about</html>")))
}
