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
)

const sourceRescanInterval = 5 * time.Minute

type sourceRescanReport struct {
	scanned            int
	added              int
	updated            int
	removed            int
	removedVariants    int
	removedArtifacts   int
	cacheInvalidations int
}

func startScheduledTasks(scheduler gocron.Scheduler, runtime *sourceRescanRuntime) error {
	started := false
	if enabled, err := registerSourceRescanTask(scheduler, runtime); err != nil {
		return err
	} else if enabled {
		started = true
	}

	if started {
		scheduler.Start()
	}
	return nil
}

func registerSourceRescanTask(scheduler gocron.Scheduler, runtime *sourceRescanRuntime) (bool, error) {
	if runtime == nil {
		return false, nil
	}

	job, err := scheduler.NewJob(
		gocron.DurationJob(sourceRescanInterval),
		gocron.NewTask(func() {
			runSourceRescan(context.Background(), runtime)
		}),
	)
	if err != nil {
		return false, fmt.Errorf("create source rescan job: %w", err)
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
		slog.Int("scanned", report.scanned),
		slog.Int("added", report.added),
		slog.Int("updated", report.updated),
		slog.Int("removed", report.removed),
		slog.Int("removed_variants", report.removedVariants),
		slog.Int("removed_artifacts", report.removedArtifacts),
		slog.Int("cache_invalidations", report.cacheInvalidations),
		slog.Duration("duration", time.Since(startedAt)),
	)
}

func syncSourceCatalog(
	ctx context.Context,
	src source.Source,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (sourceRescanReport, error) {
	report := sourceRescanReport{}
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
		report.scanned++
		return nil
	}); err != nil {
		return sourceRescanReport{}, fmt.Errorf("walk source assets: %w", err)
	}

	existingAssets := cat.AllAssets()
	existingByPath := map[string]*catalog.Asset{}
	existingAssets.Range(func(_ int, asset *catalog.Asset) bool {
		existingByPath[asset.Path] = asset
		return true
	})

	for assetPath, asset := range scannedAssets {
		existing, found := existingByPath[assetPath]
		if found {
			delete(existingByPath, assetPath)
		}

		if !found {
			report.added++
		} else if assetChanged(existing, asset) {
			report.updated++
			report.cacheInvalidations += invalidateAssetCache(bodyCache, existing.FullPath)
			report.removedVariants += removeAssetVariants(cat.DeleteVariants(assetPath), bodyCache, &report)
		}

		if err := cat.UpsertAsset(asset); err != nil {
			return sourceRescanReport{}, fmt.Errorf("upsert asset %s: %w", asset.Path, err)
		}
	}

	for _, asset := range existingByPath {
		report.removed++
		report.cacheInvalidations += invalidateAssetCache(bodyCache, asset.FullPath)
		report.removedVariants += removeAssetVariants(cat.DeleteAsset(asset.Path), bodyCache, &report)
	}

	return report, nil
}

func removeAssetVariants(
	variants collectionx.List[*catalog.Variant],
	bodyCache *assetcache.Cache,
	report *sourceRescanReport,
) int {
	removed := 0
	variants.Range(func(_ int, variant *catalog.Variant) bool {
		removed++
		if report != nil {
			report.cacheInvalidations += invalidateAssetCache(bodyCache, variant.ArtifactPath)
			report.removedArtifacts += removeArtifactFile(variant.ArtifactPath)
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
		return nil, fmt.Errorf("hash asset %s: %w", file.Path, err)
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
