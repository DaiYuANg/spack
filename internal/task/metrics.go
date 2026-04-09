package task

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
)

var (
	taskRunsTotalSpec = observabilityx.NewCounterSpec(
		"task_runs_total",
		observabilityx.WithDescription("Total number of background task executions."),
		observabilityx.WithLabelKeys("task", "result"),
	)
	taskRunDurationSpec = observabilityx.NewHistogramSpec(
		"task_run_duration_seconds",
		observabilityx.WithDescription("Background task execution duration in seconds."),
		observabilityx.WithUnit("s"),
		observabilityx.WithLabelKeys("task", "result"),
	)
	sourceRescanScannedBytesTotalSpec = observabilityx.NewCounterSpec(
		"source_rescan_scanned_bytes_total",
		observabilityx.WithDescription("Total number of scanned source bytes during source rescan tasks."),
		observabilityx.WithUnit("By"),
	)
	sourceRescanScannedTotalSpec               = observabilityx.NewCounterSpec("source_rescan_scanned_total", observabilityx.WithDescription("Total number of scanned source entries during source rescan tasks."))
	sourceRescanAddedTotalSpec                 = observabilityx.NewCounterSpec("source_rescan_added_total", observabilityx.WithDescription("Total number of newly added source entries during source rescan tasks."))
	sourceRescanUpdatedTotalSpec               = observabilityx.NewCounterSpec("source_rescan_updated_total", observabilityx.WithDescription("Total number of updated source entries during source rescan tasks."))
	sourceRescanRemovedTotalSpec               = observabilityx.NewCounterSpec("source_rescan_removed_total", observabilityx.WithDescription("Total number of removed source entries during source rescan tasks."))
	sourceRescanRemovedVariantsTotalSpec       = observabilityx.NewCounterSpec("source_rescan_removed_variants_total", observabilityx.WithDescription("Total number of removed variants during source rescan tasks."))
	sourceRescanRemovedArtifactsTotalSpec      = observabilityx.NewCounterSpec("source_rescan_removed_artifacts_total", observabilityx.WithDescription("Total number of removed artifacts during source rescan tasks."))
	sourceRescanCacheInvalidationsTotalSpec    = observabilityx.NewCounterSpec("source_rescan_cache_invalidations_total", observabilityx.WithDescription("Total number of cache invalidations triggered by source rescan tasks."))
	artifactJanitorScannedArtifactsTotalSpec   = observabilityx.NewCounterSpec("artifact_janitor_scanned_artifacts_total", observabilityx.WithDescription("Total number of artifacts scanned by janitor runs."))
	artifactJanitorRemovedOrphansTotalSpec     = observabilityx.NewCounterSpec("artifact_janitor_removed_orphans_total", observabilityx.WithDescription("Total number of orphaned artifacts removed by janitor runs."))
	artifactJanitorRemovedDirectoriesTotalSpec = observabilityx.NewCounterSpec("artifact_janitor_removed_directories_total", observabilityx.WithDescription("Total number of directories removed by janitor runs."))
	artifactJanitorMissingVariantsTotalSpec    = observabilityx.NewCounterSpec("artifact_janitor_missing_variants_total", observabilityx.WithDescription("Total number of missing variant references detected by janitor runs."))
	artifactJanitorCacheInvalidationsTotalSpec = observabilityx.NewCounterSpec("artifact_janitor_cache_invalidations_total", observabilityx.WithDescription("Total number of cache invalidations triggered by janitor runs."))
	cacheWarmerAssetsTotalSpec                 = observabilityx.NewCounterSpec("cache_warmer_assets_total", observabilityx.WithDescription("Total number of assets visited during cache warmer runs."))
	cacheWarmerVariantsTotalSpec               = observabilityx.NewCounterSpec("cache_warmer_variants_total", observabilityx.WithDescription("Total number of variants visited during cache warmer runs."))
	cacheWarmerLoadedEntriesTotalSpec          = observabilityx.NewCounterSpec("cache_warmer_loaded_entries_total", observabilityx.WithDescription("Total number of cache entries loaded by cache warmer runs."))
	cacheWarmerLoadedBytesTotalSpec            = observabilityx.NewCounterSpec("cache_warmer_loaded_bytes_total", observabilityx.WithDescription("Total number of bytes loaded by cache warmer runs."), observabilityx.WithUnit("By"))
)

func recordTaskRunMetrics(
	ctx context.Context,
	obs observabilityx.Observability,
	taskName string,
	startedAt time.Time,
	err error,
) {
	obs = observabilityx.Normalize(obs, nil)
	result := "ok"
	if err != nil {
		result = "error"
	}

	attrs := []observabilityx.Attribute{
		observabilityx.String("task", taskName),
		observabilityx.String("result", result),
	}
	obs.Counter(taskRunsTotalSpec).Add(ctx, 1, attrs...)
	obs.Histogram(taskRunDurationSpec).Record(ctx, time.Since(startedAt).Seconds(), attrs...)
}

func recordSourceRescanMetrics(ctx context.Context, obs observabilityx.Observability, report SourceRescanReport) {
	obs = observabilityx.Normalize(obs, nil)
	obs.Counter(sourceRescanScannedBytesTotalSpec).Add(ctx, report.TotalBytes)
	obs.Counter(sourceRescanScannedTotalSpec).Add(ctx, int64(report.Scanned))
	obs.Counter(sourceRescanAddedTotalSpec).Add(ctx, int64(report.Added))
	obs.Counter(sourceRescanUpdatedTotalSpec).Add(ctx, int64(report.Updated))
	obs.Counter(sourceRescanRemovedTotalSpec).Add(ctx, int64(report.Removed))
	obs.Counter(sourceRescanRemovedVariantsTotalSpec).Add(ctx, int64(report.RemovedVariants))
	obs.Counter(sourceRescanRemovedArtifactsTotalSpec).Add(ctx, int64(report.RemovedArtifacts))
	obs.Counter(sourceRescanCacheInvalidationsTotalSpec).Add(ctx, int64(report.CacheInvalidations))
}

func recordArtifactJanitorMetrics(ctx context.Context, obs observabilityx.Observability, report ArtifactJanitorReport) {
	obs = observabilityx.Normalize(obs, nil)
	obs.Counter(artifactJanitorScannedArtifactsTotalSpec).Add(ctx, int64(report.ScannedArtifacts))
	obs.Counter(artifactJanitorRemovedOrphansTotalSpec).Add(ctx, int64(report.RemovedOrphans))
	obs.Counter(artifactJanitorRemovedDirectoriesTotalSpec).Add(ctx, int64(report.RemovedDirectories))
	obs.Counter(artifactJanitorMissingVariantsTotalSpec).Add(ctx, int64(report.MissingVariants))
	obs.Counter(artifactJanitorCacheInvalidationsTotalSpec).Add(ctx, int64(report.CacheInvalidations))
}

func recordCacheWarmerMetrics(ctx context.Context, obs observabilityx.Observability, report CacheWarmerReport) {
	obs = observabilityx.Normalize(obs, nil)
	obs.Counter(cacheWarmerAssetsTotalSpec).Add(ctx, int64(report.Assets))
	obs.Counter(cacheWarmerVariantsTotalSpec).Add(ctx, int64(report.Variants))
	obs.Counter(cacheWarmerLoadedEntriesTotalSpec).Add(ctx, int64(report.LoadedEntries))
	obs.Counter(cacheWarmerLoadedBytesTotalSpec).Add(ctx, report.LoadedBytes)
}
