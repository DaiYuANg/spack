package pipeline

import (
	"context"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
)

var Module = dix.NewModule("pipeline",
	dix.WithModuleProviders(
		dix.Provider0(newMetrics),
		dix.Provider3(newImageStageFromDeps),
		dix.Provider3(newCompressionStageFromDeps),
		dix.Provider2(newStageRegistrations),
		dix.Provider1(newStages),
		dix.Provider6(newServiceFromDeps),
	),
	dix.WithModuleHooks(
		dix.OnStart(startServiceLifecycle),
		dix.OnStop(func(ctx context.Context, svc *Service) error {
			return svc.stop(ctx)
		}),
	),
)

func newStageRegistrations(image *imageStage, compression *compressionStage) collectionx.List[stageRegistration] {
	return collectionx.NewList(
		newStageRegistration(100, image),
		newStageRegistration(200, compression),
	)
}

func newStages(registrations collectionx.List[stageRegistration]) collectionx.List[Stage] {
	return buildStages(registrations)
}

func startServiceLifecycle(ctx context.Context, svc *Service) error {
	workers := max(svc.cfg.Workers, 1)
	return svc.start(ctx, workers, cap(svc.tasks))
}
