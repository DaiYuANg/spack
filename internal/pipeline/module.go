package pipeline

import (
	"context"

	"github.com/DaiYuANg/arcgo/dix"
)

var Module = dix.NewModule("pipeline",
	dix.WithModuleProviders(
		dix.Provider0(newMetrics),
		dix.Provider3(newImageStageFromDeps),
		dix.Provider3(newCompressionStageFromDeps),
		dix.Provider2(newStages),
		dix.Provider6(newServiceFromDeps),
	),
	dix.WithModuleHooks(
		dix.OnStart(startServiceLifecycle),
		dix.OnStop(func(ctx context.Context, svc *Service) error {
			return svc.stop(ctx)
		}),
	),
)

func newStages(image *imageStage, compression *compressionStage) []Stage {
	return []Stage{image, compression}
}

func startServiceLifecycle(ctx context.Context, svc *Service) error {
	workers := max(svc.cfg.Workers, 1)
	return svc.start(ctx, workers, cap(svc.tasks))
}
