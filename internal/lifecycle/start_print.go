package lifecycle

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
)

func startPrint(lc fx.Lifecycle, cfg *config.Config, logger *slog.Logger) {
	lc.Append(fx.StartHook(func() {
		localAddress := "http://127.0.0.1:" + cfg.Http.GetPort()
		accessAddress := localAddress + cfg.Spa.Path
		logger.Info("Server startup", slog.String("access path", accessAddress))
	}))
}
