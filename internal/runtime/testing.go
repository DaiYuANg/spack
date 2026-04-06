package runtime

import (
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
)

func BuildCatalogAssetForTest(file source.File) (*catalog.Asset, error) {
	return buildCatalogAsset(file)
}

func CatalogReadyAttrsForTest(
	cfg *config.Config,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	cacheStats assetcache.WarmStats,
	totalBytes int64,
	duration time.Duration,
) collectionx.List[slog.Attr] {
	return catalogReadyAttrs(cfg, cat, bodyCache, cacheStats, totalBytes, duration)
}
