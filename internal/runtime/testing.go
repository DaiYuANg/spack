package runtime

import (
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/internal/sourcecatalog"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"
)

func BuildCatalogAssetForTest(file source.File) (*catalog.Asset, error) {
	asset, err := sourcecatalog.BuildAsset(file)
	if err != nil {
		return nil, oops.In("runtime").Owner("testing").Wrap(err)
	}
	return asset, nil
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

func HTTPListenConfigForTest(cfg *config.Config) fiber.ListenConfig {
	return newHTTPListenConfig(cfg)
}
