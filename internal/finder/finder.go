package finder

import (
	"log/slog"
	"os"
	"path/filepath"

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

func (p *Finder) Lookup(ctx LookupContext) ([]byte, error) {
	p.Info("Lookup", slog.Any("path", ctx.Path))

	// 1️⃣ 尝试 Registry
	original, err := p.registry.GetOriginal(ctx.Path)
	if err == nil && original != nil {
		content, err := os.ReadFile(original.Path)
		if err == nil {
			return content, nil
		}
		// 如果读取失败，也可以 log 一下
		p.Warn("Failed to read original file, fallback will be used", slog.String("path", original.Path), slog.StringValue(err.Error()))
	}

	// 2️⃣ fallback 文件
	fallbackPath := filepath.Join(p.config.Spa.Static, p.config.Spa.Fallback)
	if fallbackPath == "" {
		return nil, err // 没有 fallback，只能返回错误
	}

	content, err := os.ReadFile(fallbackPath)
	if err != nil {
		return nil, err
	}

	return content, nil
}
