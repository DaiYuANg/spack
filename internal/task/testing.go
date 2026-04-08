package task

import (
	"context"

	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
)

// SyncSourceCatalogForTest exposes source/catalog reconciliation for black-box tests.
func SyncSourceCatalogForTest(
	ctx context.Context,
	src source.Source,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (SourceRescanReport, error) {
	return syncSourceCatalog(ctx, src, cat, bodyCache)
}

// SyncArtifactCatalogForTest exposes artifact/catalog reconciliation for black-box tests.
func SyncArtifactCatalogForTest(
	ctx context.Context,
	store artifact.Store,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (ArtifactJanitorReport, error) {
	return syncArtifactCatalog(ctx, store, cat, bodyCache)
}

// WarmCacheHotsetForTest exposes hotset warming for black-box tests.
func WarmCacheHotsetForTest(
	ctx context.Context,
	cfg *config.Config,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (CacheWarmerReport, error) {
	return warmCacheHotset(ctx, cfg, cat, bodyCache)
}
