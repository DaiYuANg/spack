package task

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/oops"
)

const sourceRescanInterval = 5 * time.Minute

type SourceRescanReport struct {
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
	report, err := syncSourceCatalog(ctx, runtime.src, runtime.catalog, runtime.bodyCache)
	if err != nil {
		runtime.logger.Error("Task source rescan failed", slog.String("err", err.Error()))
		return
	}

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
	src source.Source,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (SourceRescanReport, error) {
	scannedAssets, report, err := collectScannedAssets(ctx, src)
	if err != nil {
		return SourceRescanReport{}, err
	}

	existingByPath := indexAssetsByPath(cat.AllAssets())
	if err := reconcileScannedAssets(&report, scannedAssets, existingByPath, cat, bodyCache); err != nil {
		return SourceRescanReport{}, err
	}
	reconcileRemovedAssets(&report, existingByPath, cat, bodyCache)
	return report, nil
}

func collectScannedAssets(ctx context.Context, src source.Source) (map[string]*catalog.Asset, SourceRescanReport, error) {
	scanErr := oops.In("task").Owner("source rescan")
	report := SourceRescanReport{}
	scannedAssets := map[string]*catalog.Asset{}

	if err := src.Walk(func(file source.File) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if file.IsDir {
			return nil
		}

		asset, err := buildCatalogAsset(file)
		if err != nil {
			return err
		}
		scannedAssets[asset.Path] = asset
		report.Scanned++
		return nil
	}); err != nil {
		return nil, SourceRescanReport{}, scanErr.Wrap(err)
	}

	return scannedAssets, report, nil
}

func indexAssetsByPath(assets collectionx.List[*catalog.Asset]) map[string]*catalog.Asset {
	byPath := map[string]*catalog.Asset{}
	assets.Range(func(_ int, asset *catalog.Asset) bool {
		byPath[asset.Path] = asset
		return true
	})
	return byPath
}

func reconcileScannedAssets(
	report *SourceRescanReport,
	scannedAssets map[string]*catalog.Asset,
	existingByPath map[string]*catalog.Asset,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) error {
	for assetPath, asset := range scannedAssets {
		if err := syncScannedAsset(report, assetPath, asset, existingByPath, cat, bodyCache); err != nil {
			return err
		}
	}
	return nil
}

func syncScannedAsset(
	report *SourceRescanReport,
	assetPath string,
	asset *catalog.Asset,
	existingByPath map[string]*catalog.Asset,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) error {
	existing, found := existingByPath[assetPath]
	if found {
		delete(existingByPath, assetPath)
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
	existingByPath map[string]*catalog.Asset,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) {
	for _, asset := range existingByPath {
		report.Removed++
		invalidateAssetAndVariants(report, asset.FullPath, cat.DeleteAsset(asset.Path), bodyCache)
	}
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
			report.RemovedArtifacts += removeArtifactFile(variant.ArtifactPath)
		}
		return true
	})
	return removed
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

func buildCatalogAsset(file source.File) (*catalog.Asset, error) {
	sourceHash, err := pkg.HashFile(file.FullPath)
	if err != nil {
		return nil, oops.In("task").Owner("source rescan").With("asset_path", file.Path).Wrap(err)
	}
	return &catalog.Asset{
		Path:       file.Path,
		FullPath:   file.FullPath,
		Size:       file.Size,
		MediaType:  string(pkg.DetectMIME(file.FullPath)),
		SourceHash: sourceHash,
		ETag:       fmt.Sprintf("%q", sourceHash),
		Metadata: collectionx.NewMapFrom(map[string]string{
			"mtime_unix": strconv.FormatInt(file.ModTime.Unix(), 10),
		}),
	}, nil
}
