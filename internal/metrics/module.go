package metrics

import (
	"github.com/arl/statsviz"
	"github.com/daiyuang/spack/internal/config"
	"go.uber.org/fx"
	"log"
	"net/http"
)

var Module = fx.Module("metrics", fx.Provide(newServeMux), fx.Invoke(start))

func newServeMux() *http.ServeMux {
	return http.NewServeMux()
}

func start(lc fx.Lifecycle, mux *http.ServeMux, cfg *config.Config) error {
	err := statsviz.Register(mux)
	if err != nil {
		return err
	}
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				log.Println(http.ListenAndServe("localhost:8080", mux))
			}()
		},
		func() {},
	))
	return nil
}
