package assetcache

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/samber/hot/pkg/base"
)

func (c *Cache) readFile(path string) ([]byte, error) {
	// #nosec G304 -- path comes from resolver/catalog-selected asset paths already validated against the asset tree.
	body, err := os.ReadFile(path)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Read asset failed",
				slog.String("path", path),
				slog.String("err", err.Error()),
			)
		}
		return nil, fmt.Errorf("read resolved asset: %w", err)
	}
	return body, nil
}

func (c *Cache) recordWarmStats(ctx context.Context, stats WarmStats) {
	if stats.Entries == 0 {
		return
	}
	c.addCounterWithContext(ctx, metricAssetCacheWarmEntries, int64(stats.Entries))
	c.addCounterWithContext(ctx, metricAssetCacheWarmBytes, stats.Bytes)
}

func (c *Cache) onEviction(_ base.EvictionReason, _ string, body []byte) {
	c.addCounter(metricAssetCacheEvictions, 1)
	c.addCounter(metricAssetCacheEvictedBytes, int64(len(body)))
}

func (c *Cache) addCounter(name string, value int64) {
	c.addCounterWithContext(context.Background(), name, value)
}

func (c *Cache) addCounterWithContext(ctx context.Context, name string, value int64) {
	if value == 0 || c == nil || c.obs == nil {
		return
	}
	c.obs.AddCounter(ctx, name, value)
}
