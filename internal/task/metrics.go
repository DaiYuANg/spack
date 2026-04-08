package task

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
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
	obs.AddCounter(ctx, "task_runs_total", 1, attrs...)
	obs.RecordHistogram(ctx, "task_run_duration_seconds", time.Since(startedAt).Seconds(), attrs...)
}

func recordSourceRescanMetrics(ctx context.Context, obs observabilityx.Observability, report SourceRescanReport) {
	obs = observabilityx.Normalize(obs, nil)
	obs.AddCounter(ctx, "source_rescan_scanned_bytes_total", report.TotalBytes)
	obs.AddCounter(ctx, "source_rescan_scanned_total", int64(report.Scanned))
	obs.AddCounter(ctx, "source_rescan_added_total", int64(report.Added))
	obs.AddCounter(ctx, "source_rescan_updated_total", int64(report.Updated))
	obs.AddCounter(ctx, "source_rescan_removed_total", int64(report.Removed))
	obs.AddCounter(ctx, "source_rescan_removed_variants_total", int64(report.RemovedVariants))
	obs.AddCounter(ctx, "source_rescan_removed_artifacts_total", int64(report.RemovedArtifacts))
	obs.AddCounter(ctx, "source_rescan_cache_invalidations_total", int64(report.CacheInvalidations))
}

func recordArtifactJanitorMetrics(ctx context.Context, obs observabilityx.Observability, report ArtifactJanitorReport) {
	obs = observabilityx.Normalize(obs, nil)
	obs.AddCounter(ctx, "artifact_janitor_scanned_artifacts_total", int64(report.ScannedArtifacts))
	obs.AddCounter(ctx, "artifact_janitor_removed_orphans_total", int64(report.RemovedOrphans))
	obs.AddCounter(ctx, "artifact_janitor_removed_directories_total", int64(report.RemovedDirectories))
	obs.AddCounter(ctx, "artifact_janitor_missing_variants_total", int64(report.MissingVariants))
	obs.AddCounter(ctx, "artifact_janitor_cache_invalidations_total", int64(report.CacheInvalidations))
}

func recordCacheWarmerMetrics(ctx context.Context, obs observabilityx.Observability, report CacheWarmerReport) {
	obs = observabilityx.Normalize(obs, nil)
	obs.AddCounter(ctx, "cache_warmer_assets_total", int64(report.Assets))
	obs.AddCounter(ctx, "cache_warmer_variants_total", int64(report.Variants))
	obs.AddCounter(ctx, "cache_warmer_loaded_entries_total", int64(report.LoadedEntries))
	obs.AddCounter(ctx, "cache_warmer_loaded_bytes_total", report.LoadedBytes)
}
