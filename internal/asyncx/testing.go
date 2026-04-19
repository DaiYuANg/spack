package asyncx

import (
	"context"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/config"
)

// NewSettingsForTest exposes settings construction for external tests.
func NewSettingsForTest(cfg *config.Async) *Settings {
	return newSettings(cfg)
}

// NewRuntimeMetricsForTest exposes runtime metrics construction for external tests.
func NewRuntimeMetricsForTest(settings *Settings) *RuntimeMetrics {
	return NewRuntimeMetrics(settings)
}

// RunListForTest exposes RunList with explicit observability metadata for external tests.
func RunListForTest[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	settings *Settings,
	workload string,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	return RunList(ctx, obs, settings, workload, values, run)
}
