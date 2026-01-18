package http

import (
	"log/slog"
	"strings"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/finder"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type SpaMiddlewareDependency struct {
	fx.In
	App               *fiber.App
	Config            *config.Config
	Log               *slog.Logger
	HttpRequestsTotal *prometheus.CounterVec
	Finder            *finder.Finder
}

func spaMiddleware(dep SpaMiddlewareDependency) {
	app, cfg, logger, total, f := dep.App, dep.Config, dep.Log, dep.HttpRequestsTotal, dep.Finder

	servePath := strings.TrimSpace(cfg.Assets.Path) + "*"
	app.Use(servePath, func(c *fiber.Ctx) error {
		// ---- 计数器辅助函数 ----
		incr := func(label string) {
			total.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}

		// ---- 处理请求路径 ----
		reqPath := strings.TrimPrefix(c.Path(), "/")          // 去掉前导 /
		spaPrefix := strings.TrimPrefix(cfg.Assets.Path, "/") // 去掉 SPA 前缀
		lookupPath := strings.TrimPrefix(reqPath, spaPrefix)
		lookupPath = strings.TrimPrefix(lookupPath, "/") // 保证无多余 /

		logger.Debug("SPA request path", slog.String("reqPath", reqPath))
		logger.Debug("SPA lookup path", slog.String("lookupPath", lookupPath))

		// ---- 查找文件 ----
		result, err := f.Lookup(finder.NewLookupContext(c.Get("Accept-Encoding"), lookupPath))
		if err != nil {
			logger.Debug("Lookup failed, trying fallback", slog.Any("error", oops.Wrap(err)))
			incr("not_found")

			if cfg.Assets.Fallback.On == config.FallbackOnNotFound && cfg.Assets.Fallback.Target != "" {
				result, err = f.Lookup(finder.NewLookupContext("", strings.TrimPrefix(cfg.Assets.Fallback.Target, "/")))
				if err != nil {
					logger.Error("Fallback lookup failed", slog.Any("error", oops.Wrap(err)))
					return fiber.ErrNotFound
				}
			} else {
				return fiber.ErrNotFound
			}
		} else {
			incr("hit")
		}

		// ---- 返回结果 ----
		c.Set(fiber.HeaderContentType, result.MediaTypeString())
		return c.Send(result.Data)
	})
}
