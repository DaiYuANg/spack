package finder

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

type Finder struct {
	registry registry.Registry
	*slog.Logger
	httpRequestsTotal *prometheus.CounterVec
	config            *config.Config
}

type Dependency struct {
	fx.In
	Config            *config.Config
	Log               *slog.Logger
	HttpRequestsTotal *prometheus.CounterVec
	Registry          registry.Registry
}

func newFinder(dependency Dependency) *Finder {
	return &Finder{
		Logger:            dependency.Log,
		registry:          dependency.Registry,
		httpRequestsTotal: dependency.HttpRequestsTotal,
		config:            dependency.Config,
	}
}
