package task

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/go-co-op/gocron/v2"
	"github.com/samber/oops"
)

const artifactJanitorInterval = 15 * time.Minute

type ArtifactJanitorReport struct {
	ScannedArtifacts   int
	RemovedOrphans     int
	RemovedDirectories int
	MissingVariants    int
	CacheInvalidations int
}

func registerArtifactJanitorTask(ctx context.Context, scheduler gocron.Scheduler, runtime *artifactJanitorRuntime) (bool, error) {
	if runtime == nil || runtime.store == nil || strings.TrimSpace(runtime.store.Root()) == "" {
		return false, nil
	}

	job, err := scheduler.NewJob(
		gocron.DurationJob(artifactJanitorInterval),
		gocron.NewTask(func() {
			runArtifactJanitor(ctx, runtime)
		}),
		gocron.WithName("artifact_janitor"),
	)
	if err != nil {
		return false, oops.In("task").Owner("artifact janitor").Wrap(err)
	}

	runtime.logger.Info("Task artifact janitor enabled",
		slog.String("id", job.ID().String()),
		slog.String("interval", artifactJanitorInterval.String()),
		slog.String("root", runtime.store.Root()),
	)
	return true, nil
}

func runArtifactJanitor(ctx context.Context, runtime *artifactJanitorRuntime) {
	startedAt := time.Now()
	report, err := syncArtifactCatalog(ctx, runtime.store, runtime.catalog, runtime.bodyCache)
	recordTaskRunMetrics(ctx, runtime.obs, "artifact_janitor", startedAt, err)
	if err != nil {
		runtime.logger.Error("Task artifact janitor failed", slog.String("err", err.Error()))
		return
	}
	recordArtifactJanitorMetrics(ctx, runtime.obs, report)
	runtime.catMetrics.SyncCatalog(runtime.catalog)
	if report.ScannedArtifacts == 0 && report.RemovedOrphans == 0 && report.MissingVariants == 0 && report.RemovedDirectories == 0 {
		return
	}

	runtime.logger.Info("Task artifact janitor completed",
		slog.Int("scanned_artifacts", report.ScannedArtifacts),
		slog.Int("removed_orphans", report.RemovedOrphans),
		slog.Int("missing_variants", report.MissingVariants),
		slog.Int("removed_directories", report.RemovedDirectories),
		slog.Int("cache_invalidations", report.CacheInvalidations),
		slog.Duration("duration", time.Since(startedAt)),
	)
}

func syncArtifactCatalog(
	ctx context.Context,
	store artifact.Store,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (ArtifactJanitorReport, error) {
	root := strings.TrimSpace(store.Root())
	if root == "" {
		return ArtifactJanitorReport{}, nil
	}

	expected, report, err := collectCatalogArtifactPaths(ctx, cat, bodyCache)
	if err != nil {
		return ArtifactJanitorReport{}, err
	}
	if err := removeOrphanArtifacts(ctx, root, expected, cat, bodyCache, &report); err != nil {
		return ArtifactJanitorReport{}, err
	}

	report.RemovedDirectories += pruneEmptyArtifactDirs(root)
	return report, nil
}

func collectCatalogArtifactPaths(
	ctx context.Context,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) (collectionx.Set[string], ArtifactJanitorReport, error) {
	expected := collectionx.NewSet[string]()
	report := ArtifactJanitorReport{}
	for _, entry := range cat.Snapshot().Assets.Values() {
		if err := janitorContextErr(ctx); err != nil {
			return nil, ArtifactJanitorReport{}, err
		}
		if err := collectCatalogEntry(entry, cat, bodyCache, expected, &report); err != nil {
			return nil, ArtifactJanitorReport{}, err
		}
	}
	return expected, report, nil
}

func collectCatalogEntry(
	entry *catalog.Entry,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	expected collectionx.Set[string],
	report *ArtifactJanitorReport,
) error {
	if entry == nil || entry.Asset == nil {
		return nil
	}
	for _, variant := range entry.Variants.Values() {
		if err := collectCatalogVariant(variant, cat, bodyCache, expected, report); err != nil {
			return err
		}
	}
	return nil
}

func collectCatalogVariant(
	variant *catalog.Variant,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	expected collectionx.Set[string],
	report *ArtifactJanitorReport,
) error {
	artifactPath := variantArtifactPath(variant)
	if artifactPath == "" {
		return nil
	}
	if artifactExists(artifactPath) {
		expected.Add(artifactPath)
		return nil
	}
	if _, statErr := os.Stat(artifactPath); statErr != nil && !os.IsNotExist(statErr) {
		return oops.In("task").Owner("artifact janitor").With("artifact_path", artifactPath).Wrap(statErr)
	}
	if cat.DeleteVariantByArtifactPath(artifactPath) {
		report.MissingVariants++
		report.CacheInvalidations += invalidateAssetCache(bodyCache, artifactPath)
	}
	return nil
}

func removeOrphanArtifacts(
	ctx context.Context,
	root string,
	expected collectionx.Set[string],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	report *ArtifactJanitorReport,
) error {
	rootHandle, err := openArtifactRoot(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if rootHandle == nil {
		return nil
	}

	walkErr := walkOrphanArtifacts(ctx, rootHandle, root, expected, cat, bodyCache, report)
	closeErr := closeRoot(rootHandle)
	if walkErr != nil {
		return walkErr
	}
	return closeErr
}

func removeOrphanArtifact(
	root *os.Root,
	relativePath string,
	artifactPath string,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	report *ArtifactJanitorReport,
) error {
	if removeErr := root.Remove(filepath.ToSlash(relativePath)); removeErr != nil && !os.IsNotExist(removeErr) {
		return oops.In("task").Owner("artifact janitor").With("artifact_path", artifactPath).Wrap(removeErr)
	}
	if report != nil {
		report.RemovedOrphans++
		report.CacheInvalidations += invalidateAssetCache(bodyCache, artifactPath)
	}
	cat.DeleteVariantByArtifactPath(artifactPath)
	return nil
}

func walkOrphanArtifacts(
	ctx context.Context,
	rootHandle *os.Root,
	root string,
	expected collectionx.Set[string],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	report *ArtifactJanitorReport,
) error {
	walkErr := fs.WalkDir(rootHandle.FS(), ".", func(relativePath string, entry fs.DirEntry, err error) error {
		return visitArtifactPath(ctx, rootHandle, root, relativePath, entry, err, expected, cat, bodyCache, report)
	})
	if walkErr == nil || os.IsNotExist(walkErr) {
		return nil
	}
	return oops.In("task").Owner("artifact janitor").Wrap(walkErr)
}

func pruneEmptyArtifactDirs(root string) int {
	if strings.TrimSpace(root) == "" {
		return 0
	}

	directories := collectionx.NewList[string]()
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root {
			directories.Add(path)
		}
		return nil
	}); err != nil {
		return 0
	}

	removed := 0
	directories.Sort(func(left, right string) int {
		return len(filepath.Clean(right)) - len(filepath.Clean(left))
	}).Range(func(_ int, path string) bool {
		if err := os.Remove(path); err == nil {
			removed++
		}
		return true
	})
	return removed
}
