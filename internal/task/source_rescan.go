package task

import (
	"cmp"
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/sourcecatalog"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/oops"
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

func registerSourceRescanTask(ctx context.Context, scheduler gocron.Scheduler, runtime *sourceRescanRuntime) (bool, error) {
	if runtime == nil {
		return false, nil
	}

	job, err := scheduler.NewJob(
		gocron.DurationJob(sourceRescanInterval),
		gocron.NewTask(func() {
			runSourceRescan(ctx, runtime)
		}),
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
	startedAt := time.Now()
	report, err := syncSourceCatalog(ctx, runtime.scanner, runtime.catalog, runtime.bodyCache)
	recordTaskRunMetrics(ctx, runtime.obs, "source_rescan", startedAt, err)
	if err != nil {
		runtime.logger.Error("Task source rescan failed", slog.String("err", err.Error()))
		return
	}
	recordSourceRescanMetrics(ctx, runtime.obs, report)
	runtime.catMetrics.SyncCatalog(runtime.catalog)
	runtime.catMetrics.SetSourceBytes(report.TotalBytes)

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
	snapshot, report, err := collectScannedSnapshot(ctx, scanner)
	if err != nil {
		return SourceRescanReport{}, err
	}

	existingByPath := indexAssetsByPath(cat.AllAssets())
	if err := reconcileScannedAssets(&report, snapshot.Assets, existingByPath, cat, bodyCache); err != nil {
		return SourceRescanReport{}, err
	}
	reconcileRemovedAssets(&report, existingByPath, cat, bodyCache)
	if err := reconcileSourceSidecars(&report, snapshot.Variants, cat, bodyCache); err != nil {
		return SourceRescanReport{}, err
	}
	return report, nil
}

func collectScannedSnapshot(ctx context.Context, scanner sourcecatalog.Scanner) (sourcecatalog.Snapshot, SourceRescanReport, error) {
	scanErr := oops.In("task").Owner("source rescan")
	snapshot, err := scanner.Scan(ctx)
	if err != nil {
		return sourcecatalog.Snapshot{}, SourceRescanReport{}, scanErr.Wrap(err)
	}
	return snapshot, SourceRescanReport{
		TotalBytes: snapshot.TotalBytes,
		Scanned:    snapshot.Assets.Len(),
	}, nil
}

func indexAssetsByPath(assets collectionx.List[*catalog.Asset]) collectionx.Map[string, *catalog.Asset] {
	byPath := collectionx.NewMapWithCapacity[string, *catalog.Asset](assets.Len())
	assets.Range(func(_ int, asset *catalog.Asset) bool {
		byPath.Set(asset.Path, asset)
		return true
	})
	return byPath
}

func reconcileScannedAssets(
	report *SourceRescanReport,
	scannedAssets collectionx.Map[string, *catalog.Asset],
	existingByPath collectionx.Map[string, *catalog.Asset],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) error {
	var syncErr error
	collectionx.NewList(scannedAssets.Keys()...).Sort(cmp.Compare[string]).Range(func(_ int, assetPath string) bool {
		asset, _ := scannedAssets.Get(assetPath)
		if err := syncScannedAsset(report, assetPath, asset, existingByPath, cat, bodyCache); err != nil {
			syncErr = err
			return false
		}
		return true
	})
	return syncErr
}

func syncScannedAsset(
	report *SourceRescanReport,
	assetPath string,
	asset *catalog.Asset,
	existingByPath collectionx.Map[string, *catalog.Asset],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) error {
	existing, found := existingByPath.Get(assetPath)
	if found {
		existingByPath.Delete(assetPath)
	}

	if !found {
		report.Added++
	} else if assetChanged(existing, asset) {
		report.Updated++
		invalidateAssetAndVariants(report, existing.FullPath, cat.DeleteVariants(assetPath), bodyCache)
	}

	if err := cat.UpsertAsset(asset); err != nil {
		return oops.In("task").Owner("source rescan").With("asset_path", asset.Path).Wrap(err)
	}
	return nil
}

func reconcileRemovedAssets(
	report *SourceRescanReport,
	existingByPath collectionx.Map[string, *catalog.Asset],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) {
	collectionx.NewList(existingByPath.Values()...).Sort(func(left, right *catalog.Asset) int {
		return cmp.Compare(left.Path, right.Path)
	}).Range(func(_ int, asset *catalog.Asset) bool {
		report.Removed++
		invalidateAssetAndVariants(report, asset.FullPath, cat.DeleteAsset(asset.Path), bodyCache)
		return true
	})
}

func reconcileSourceSidecars(
	report *SourceRescanReport,
	scannedVariants collectionx.Map[string, *catalog.Variant],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) error {
	existingByID := indexSourceSidecarVariants(cat)
	var syncErr error
	collectionx.NewList(scannedVariants.Keys()...).Sort(cmp.Compare[string]).Range(func(_ int, variantID string) bool {
		variant, _ := scannedVariants.Get(variantID)
		if err := cat.UpsertVariant(variant); err != nil {
			syncErr = oops.In("task").Owner("source rescan").With("variant_id", variantID).With("asset_path", variant.AssetPath).Wrap(err)
			return false
		}
		existingByID.Delete(variantID)
		return true
	})
	if syncErr != nil {
		return syncErr
	}

	collectionx.NewList(existingByID.Values()...).Sort(func(left, right *catalog.Variant) int {
		return cmp.Compare(left.ID, right.ID)
	}).Range(func(_ int, variant *catalog.Variant) bool {
		if !cat.DeleteVariantByArtifactPath(variant.ArtifactPath) {
			return true
		}
		report.RemovedVariants++
		report.CacheInvalidations += invalidateAssetCache(bodyCache, variant.ArtifactPath)
		return true
	})
	return nil
}

func indexSourceSidecarVariants(cat catalog.Catalog) collectionx.Map[string, *catalog.Variant] {
	variantsByID := collectionx.NewMap[string, *catalog.Variant]()
	cat.AllAssets().Range(func(_ int, asset *catalog.Asset) bool {
		cat.ListVariants(asset.Path).Range(func(_ int, variant *catalog.Variant) bool {
			if sourcecatalog.IsSourceSidecarVariant(variant) {
				variantsByID.Set(variant.ID, variant)
			}
			return true
		})
		return true
	})
	return variantsByID
}

func invalidateAssetAndVariants(
	report *SourceRescanReport,
	assetPath string,
	variants collectionx.List[*catalog.Variant],
	bodyCache *assetcache.Cache,
) {
	if report == nil {
		return
	}
	report.CacheInvalidations += invalidateAssetCache(bodyCache, assetPath)
	report.RemovedVariants += removeAssetVariants(variants, bodyCache, report)
}

func removeAssetVariants(
	variants collectionx.List[*catalog.Variant],
	bodyCache *assetcache.Cache,
	report *SourceRescanReport,
) int {
	removed := 0
	variants.Range(func(_ int, variant *catalog.Variant) bool {
		removed++
		if report != nil {
			report.CacheInvalidations += invalidateAssetCache(bodyCache, variant.ArtifactPath)
			report.RemovedArtifacts += removeVariantArtifact(variant)
		}
		return true
	})
	return removed
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
