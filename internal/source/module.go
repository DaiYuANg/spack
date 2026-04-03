package source

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("source",
	dix.WithModuleSetups(
		dix.SetupWithMetadata(setupSource, dix.SetupMetadata{
			Label: "SetupSource",
			Dependencies: dix.ServiceRefs(
				dix.TypedService[*config.Assets](),
				dix.TypedService[*slog.Logger](),
			),
			Provides: dix.ServiceRefs(
				dix.TypedService[Source](),
			),
			GraphMutation: true,
		}),
	),
)

func setupSource(c *dix.Container, _ dix.Lifecycle) error {
	cfg, err := dix.ResolveAs[*config.Assets](c)
	if err != nil {
		return err
	}
	logger, err := dix.ResolveAs[*slog.Logger](c)
	if err != nil {
		return err
	}

	src, err := newLocalFS(cfg, logger)
	if err != nil {
		return err
	}
	dix.ProvideValueT[Source](c, src)
	return nil
}
