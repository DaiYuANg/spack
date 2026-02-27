package http

import (
	"path/filepath"
	"strings"

	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

func requestMeta(cfg *config.Config, httpRequestsTotal *prometheus.CounterVec) fiber.Handler {
	return func(c fiber.Ctx) error {
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Assets.Root, reqPath)

		// 将请求路径挂到 Context 上，供后续中间件使用
		c.Locals("finder:fullPath", fullPath)
		c.Locals("finder:reqPath", reqPath)

		// 提供一个统一的打点函数
		incr := func(label string) {
			httpRequestsTotal.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}
		c.Locals("finder:incr", incr)

		return c.Next()
	}

}
