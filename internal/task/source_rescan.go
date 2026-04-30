package task

import (
	"cmp"
	"context"
	cxlist "github.com/arcgolabs/collectionx/list"
	cxmapping "github.com/arcgolabs/collectionx/mapping"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/sourcecatalog"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/oops"
	"log/slog"
	"os"
	"time"
)

const sourceRescanInterval = 5 * time.Minute

type SourceRescanReport struct {
	TotalBytes         int64
	Scanned            int
	Added              int
	Updated            int
	Removed            int
	RemovedVariants    int
	RemovedArtifacts   int
	CacheInvalidations int
}

type sourceRescanRun struct {
	ctx       context.Context
	scanner   sourcecatalog.Scanner
	cat       catalog.Catalog
	bodyCache *assetcache.Cache
	report    SourceRescanReport
}

func registerSourceRescanTask(ctx context.Context, scheduler gocron.Scheduler, runtime *sourceRescanRuntime) (bool, error) {
	if runtime == nil {
		return false, nil
	}

	job, err := scheduler.NewJob(
		gocron.DurationJob(sourceRescanInterval),
		gocron.NewTask(func() {
			runSourceRescan(ctx, runtime)
		}),
		gocron.WithName("source_rescan"),
	)
	if err != nil {
		return false, oops.In("task").Owner("source rescan").Wrap(err)
	}

	runtime.logger.Info("Task source rescan enabled",
		slog.String("id", job.ID().String()),
		slog.String("interval", sourceRescanInterval.String()),
	)
	return true, nil
}

func runSourceRescan(ctx context.Context, runtime *sourceRescanRuntime) {
	runtime.rescanMu.Lock()
	defer runtime.rescanMu.Unlock()

	startedAt := time.Now()
	report, err := syncSourceCatalog(ctx, runtime.scanner, runtime.catalog, runtime.bodyCache)
	recordTaskRunMetrics(ctx, runtime.obs, "source_rescan", startedAt, err)
	if err != nil {
		runtime.logger.Error("Task source rescan failed", slog.String("err", err.Error()))
		return
	}
	recordSourceRescanMetrics(ctx, runtime.obs, report)
	go runtime.catMetrics.SyncCatalog(runtime.catalog)
	go runtime.catMetrics.SetSourceBytes(report.TotalBytes)

	runtime.logger.Info("Task source rescan completed",
		slog.Int("scanned", report.Scanned),
		slog.Int("added", report.Added),
		slog.Int("updated", report.Updated),
		slog.Int("removed", report.Removed),
		slog.Int("removed_variants", report.RemovedVariants),
		slog.Int("removed_artifacts", report.RemovedArtifacts),
		slog.Int("cache_invalidations", report.CacheInvalidations),
		slog.Duration("duration", time.Since(startedAt)),
	)
}

func syncSourceCatalog(
	ctx context.Context,
	scanner sourcecatalog.Scanner,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (SourceRescanReport, error) {
	run := sourceRescanRun{
		ctx:       ctx,
		scanner:   scanner,
		cat:       cat,
		bodyCache: bodyCache,
	}
	return run.sync()
}

func (r *sourceRescanRun) sync() (SourceRescanReport, error) {
	snapshot, err := r.collectScannedSnapshot()
	if err != nil {
		return SourceRescanReport{}, err
	}

	existingByPath := indexAssetsByPath(r.cat.AllAssets())
	if err := r.reconcileScannedAssets(snapshot.Assets, existingByPath); err != nil {
		return SourceRescanReport{}, err
	}
	r.reconcileRemovedAssets(existingByPath)
	if err := r.reconcileSourceSidecars(snapshot.Variants); err != nil {
		return SourceRescanReport{}, err
	}
	return r.report, nil
}

func (r *sourceRescanRun) collectScannedSnapshot() (sourcecatalog.Snapshot, error) {
	scanErr := oops.In("task").Owner("source rescan")
	snapshot, err := r.scanner.ScanWithCatalog(r.ctx, r.cat)
	if err != nil {
		return sourcecatalog.Snapshot{}, scanErr.Wrap(err)
	}
	r.report.TotalBytes = snapshot.TotalBytes
	r.report.Scanned = snapshot.Assets.Len()
	return snapshot, nil
}

func indexAssetsByPath(assets *cxlist.List[*catalog.Asset]) *cxmapping.Map[string, *catalog.Asset] {
	return cxmapping.AssociateList[*catalog.Asset, string, *catalog.Asset](assets, func(_ int, asset *catalog.Asset) (string, *catalog.Asset) {
		return asset.Path, asset
	})
}

func (r *sourceRescanRun) reconcileScannedAssets(
	scannedAssets *cxmapping.Map[string, *catalog.Asset],
	existingByPath *cxmapping.Map[string, *catalog.Asset],
) error {
	var syncErr error
	sortedMapKeys[*catalog.Asset](scannedAssets).Range(func(_ int, assetPath string) bool {
		asset, _ := scannedAssets.Get(assetPath)
		if err := r.syncScannedAsset(assetPath, asset, existingByPath); err != nil {
			syncErr = err
			return false
		}
		return true
	})
	return syncErr
}

func (r *sourceRescanRun) syncScannedAsset(
	assetPath string,
	asset *catalog.Asset,
	existingByPath *cxmapping.Map[string, *catalog.Asset],
) error {
	existing, found := existingByPath.Get(assetPath)
	if found {
		existingByPath.Delete(assetPath)
	}

	if !found {
		r.report.Added++
	} else if assetChanged(existing, asset) {
		r.report.Updated++
		r.invalidateAssetAndVariants(existing.FullPath, r.cat.DeleteVariants(assetPath))
	}

	if err := r.cat.UpsertAsset(asset); err != nil {
		return oops.In("task").Owner("source rescan").With("asset_path", asset.Path).Wrap(err)
	}
	return nil
}

func (r *sourceRescanRun) reconcileRemovedAssets(
	existingByPath *cxmapping.Map[string, *catalog.Asset],
) {
	cxlist.NewList[*catalog.Asset](existingByPath.Values()...).Sort(func(left, right *catalog.Asset) int {
		return cmp.Compare(left.Path, right.Path)
	}).Range(func(_ int, asset *catalog.Asset) bool {
		r.report.Removed++
		r.invalidateAssetAndVariants(asset.FullPath, r.cat.DeleteAsset(asset.Path))
		return true
	})
}

func (r *sourceRescanRun) reconcileSourceSidecars(
	scannedVariants *cxmapping.Map[string, *catalog.Variant],
) error {
	existingByID := r.indexSourceSidecarVariants()
	var syncErr error
	sortedMapKeys[*catalog.Variant](scannedVariants).Range(func(_ int, variantID string) bool {
		variant, _ := scannedVariants.Get(variantID)
		if err := r.cat.UpsertVariant(variant); err != nil {
			syncErr = oops.In("task").Owner("source rescan").With("variant_id", variantID).With("asset_path", variant.AssetPath).Wrap(err)
			return false
		}
		existingByID.Delete(variantID)
		return true
	})
	if syncErr != nil {
		return syncErr
	}

	cxlist.NewList[*catalog.Variant](existingByID.Values()...).Sort(func(left, right *catalog.Variant) int {
		return cmp.Compare(left.ID, right.ID)
	}).Range(func(_ int, variant *catalog.Variant) bool {
		if !r.cat.DeleteVariantByArtifactPath(variant.ArtifactPath) {
			return true
		}
		r.report.RemovedVariants++
		r.report.CacheInvalidations += invalidateAssetCache(r.bodyCache, variant.ArtifactPath)
		return true
	})
	return nil
}

func (r *sourceRescanRun) indexSourceSidecarVariants() *cxmapping.Map[string, *catalog.Variant] {
	variantsByID := cxmapping.NewMap[string, *catalog.Variant]()
	r.cat.ListVariantsByStage(sourcecatalog.SourceSidecarStage).Range(func(_ int, variant *catalog.Variant) bool {
		if sourcecatalog.IsSourceSidecarVariant(variant) {
			variantsByID.Set(variant.ID, variant)
		}
		return true
	})
	return variantsByID
}

func (r *sourceRescanRun) invalidateAssetAndVariants(
	assetPath string,
	variants *cxlist.List[*catalog.Variant],
) {
	r.report.CacheInvalidations += invalidateAssetCache(r.bodyCache, assetPath)
	r.report.RemovedVariants += r.removeAssetVariants(variants)
}

func (r *sourceRescanRun) removeAssetVariants(variants *cxlist.List[*catalog.Variant]) int {
	removed := 0
	variants.Range(func(_ int, variant *catalog.Variant) bool {
		removed++
		r.report.CacheInvalidations += invalidateAssetCache(r.bodyCache, variant.ArtifactPath)
		r.report.RemovedArtifacts += removeVariantArtifact(variant)
		return true
	})
	return removed
}

func sortedMapKeys[T any](values *cxmapping.Map[string, T]) *cxlist.List[string] {
	return cxlist.NewList[string](values.Keys()...).Sort(cmp.Compare[string])
}

func removeVariantArtifact(variant *catalog.Variant) int {
	if sourcecatalog.IsSourceSidecarVariant(variant) {
		return 0
	}
	return removeArtifactFile(variant.ArtifactPath)
}

func invalidateAssetCache(bodyCache *assetcache.Cache, path string) int {
	if bodyCache != nil && bodyCache.Delete(path) {
		return 1
	}
	return 0
}

func removeArtifactFile(path string) int {
	if path == "" {
		return 0
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return 0
	}
	return 1
}

func assetChanged(existing, next *catalog.Asset) bool {
	if existing == nil || next == nil {
		return existing != next
	}
	return existing.FullPath != next.FullPath ||
		existing.Size != next.Size ||
		existing.MediaType != next.MediaType ||
		existing.SourceHash != next.SourceHash ||
		existing.ETag != next.ETag
}
