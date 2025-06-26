package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"path/filepath"
	"sproxy/internal/config"
	"strings"
)

func requestMeta(cfg *config.Config, httpRequestsTotal *prometheus.CounterVec) fiber.Handler {
	return func(c fiber.Ctx) error {
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Spa.Static, reqPath)

		// 将请求路径挂到 Context 上，供后续中间件使用
		c.Locals("spa:fullPath", fullPath)
		c.Locals("spa:reqPath", reqPath)

		// 提供一个统一的打点函数
		incr := func(label string) {
			httpRequestsTotal.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}
		c.Locals("spa:incr", incr)

		return c.Next()
	}

}
