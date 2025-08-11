package spa

import (
	"path/filepath"
	"strings"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Processor struct {
	registry          registry.Registry
	logger            *zap.SugaredLogger
	httpRequestsTotal *prometheus.CounterVec
	config            *config.Config
}

type ProcessorDependency struct {
	fx.In
	Config            *config.Config
	Log               *zap.SugaredLogger
	HttpRequestsTotal *prometheus.CounterVec
	Registry          registry.Registry
}

func NewProcessor(dependency ProcessorDependency) *Processor {
	return &Processor{
		registry:          dependency.Registry,
		logger:            dependency.Log,
		httpRequestsTotal: dependency.HttpRequestsTotal,
		config:            dependency.Config,
	}
}

func (p *Processor) Handle() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_ = func(label string) {
			p.httpRequestsTotal.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}
		reqPath := strings.TrimPrefix(c.Path(), "/")
		_ = filepath.Join(p.config.Spa.Static, reqPath)
		//incr(constant.Compress)
		original, err := p.registry.GetOriginal(reqPath)
		if err != nil {
			return err
		}
		return c.SendFile(original.Path)
	}
}
