package workerpool

import (
	"context"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/config"
	"github.com/panjf2000/ants/v2"
)

// NewSettingsForTest exposes settings construction for external tests.
func NewSettingsForTest(cfg *config.Async) *Settings {
	return newSettings(cfg)
}

// NewPoolForTest exposes shared pool construction for external tests.
func NewPoolForTest(settings *Settings) (*ants.Pool, error) {
	return newPool(settings)
}

// NewRuntimeMetricsForTest exposes runtime metrics construction for external tests.
func NewRuntimeMetricsForTest(settings *Settings, pool *ants.Pool) *RuntimeMetrics {
	return NewRuntimeMetrics(settings, pool)
}

// RunListForTest exposes RunList with explicit observability metadata for external tests.
func RunListForTest[T any](
	ctx context.Context,
	obs observabilityx.Observability,
	pool *ants.Pool,
	workload string,
	values collectionx.List[T],
	run func(context.Context, T) error,
) error {
	return RunList(ctx, obs, pool, workload, values, run)
}
