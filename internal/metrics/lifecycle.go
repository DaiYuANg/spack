package metrics

import (
	"net/http"

	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func start(lc fx.Lifecycle, mux *http.ServeMux, cfg *config.Config, logger *zap.SugaredLogger) error {
	if !cfg.Debug.Enable {
		return nil
	}
	err := statsviz.Register(mux)
	if err != nil {
		return err
	}
	lc.Append(fx.StartHook(
		func() {
			go func() {
				logger.Infof("Metrics server start:%s", "http://localhost:8080/debug/statsviz")
				_ = http.ListenAndServe("localhost:8080", mux)
			}()
		},
	))
	return nil
}
