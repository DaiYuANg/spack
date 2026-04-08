package task

import (
	"context"
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
	if err != nil {
		runtime.logger.Error("Task artifact janitor failed", slog.String("err", err.Error()))
		return
	}
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
	entries := cat.Snapshot().Assets

	var collectErr error
	entries.Range(func(_ int, entry *catalog.Entry) bool {
		if err := ctx.Err(); err != nil {
			collectErr = err
			return false
		}
		if entry == nil || entry.Asset == nil {
			return true
		}

		entry.Variants.Range(func(_ int, variant *catalog.Variant) bool {
			if err := ctx.Err(); err != nil {
				collectErr = err
				return false
			}
			if variant == nil || strings.TrimSpace(variant.ArtifactPath) == "" {
				return true
			}

			artifactPath := variant.ArtifactPath
			if _, statErr := os.Stat(artifactPath); statErr != nil {
				if os.IsNotExist(statErr) {
					if cat.DeleteVariantByArtifactPath(artifactPath) {
						report.MissingVariants++
						report.CacheInvalidations += invalidateAssetCache(bodyCache, artifactPath)
					}
					return true
				}
				collectErr = oops.In("task").Owner("artifact janitor").With("artifact_path", artifactPath).Wrap(statErr)
				return false
			}

			expected.Add(artifactPath)
			return true
		})
		return collectErr == nil
	})

	if collectErr != nil {
		return nil, ArtifactJanitorReport{}, collectErr
	}
	return expected, report, nil
}

func removeOrphanArtifacts(
	ctx context.Context,
	root string,
	expected collectionx.Set[string],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	report *ArtifactJanitorReport,
) error {
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if walkCtxErr := ctx.Err(); walkCtxErr != nil {
			return walkCtxErr
		}
		if entry.IsDir() {
			return nil
		}

		if report != nil {
			report.ScannedArtifacts++
		}
		if expected != nil && expected.Contains(path) {
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return oops.In("task").Owner("artifact janitor").With("artifact_path", path).Wrap(removeErr)
		}

		if report != nil {
			report.RemovedOrphans++
			report.CacheInvalidations += invalidateAssetCache(bodyCache, path)
		}
		cat.DeleteVariantByArtifactPath(path)
		return nil
	})
	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return nil
		}
		return oops.In("task").Owner("artifact janitor").Wrap(walkErr)
	}
	return nil
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
