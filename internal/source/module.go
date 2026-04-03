package source

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/config"
)

var Module = dix.NewModule("source",
	dix.WithModuleSetups(
		dix.Setup(setupSource),
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
