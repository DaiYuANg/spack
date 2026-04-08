package task

import (
	"context"

	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
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
