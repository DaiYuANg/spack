package pipeline

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("pipeline",
	dix.WithModuleProviders(
		dix.Provider0(newMetrics),
		dix.Provider3(newImageStageFromDeps),
		dix.Provider3(newCompressionStageFromDeps),
		dix.Provider2(newStages),
	),
	dix.WithModuleSetups(
		dix.Setup(setupService),
	),
)

func newStages(image *imageStage, compression *compressionStage) []Stage {
	return []Stage{image, compression}
}

func setupService(c *dix.Container, lc dix.Lifecycle) error {
	cfg, err := dix.ResolveAs[*config.Compression](c)
	if err != nil {
		return err
	}
	logger, err := dix.ResolveAs[*slog.Logger](c)
	if err != nil {
		return err
	}
	cat, err := dix.ResolveAs[catalog.Catalog](c)
	if err != nil {
		return err
	}
	metrics, err := dix.ResolveAs[*Metrics](c)
	if err != nil {
		return err
	}
	stages, err := dix.ResolveAs[[]Stage](c)
	if err != nil {
		return err
	}

	dix.ProvideValueT(c, newService(lc, cfg, logger, cat, metrics, stages))
	return nil
}
