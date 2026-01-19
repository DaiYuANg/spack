package http

import (
	"log/slog"
	"strings"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/finder"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/oops"
	goeventbus "github.com/stanipetrosyan/go-eventbus"
	"go.uber.org/fx"
)

type AssetsMiddlewareDependency struct {
	fx.In
	App               *fiber.App
	Config            *config.Config
	Log               *slog.Logger
	HttpRequestsTotal *prometheus.CounterVec
	Finder            *finder.Finder
	EventBus          goeventbus.EventBus
}

func assetsMiddleware(dep AssetsMiddlewareDependency) {
	app, cfg, logger, total, f := dep.App, dep.Config, dep.Log, dep.HttpRequestsTotal, dep.Finder

	servePath := strings.TrimSpace(cfg.Assets.Path) + "*"
	app.Use(servePath, func(c *fiber.Ctx) error {
		// ---- 计数器辅助函数 ----
		incr := func(label string) {
			total.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}

		// ---- 处理请求路径 ----
		reqPath := strings.TrimPrefix(c.Path(), "/")             // 去掉前导 /
		assetsPrefix := strings.TrimPrefix(cfg.Assets.Path, "/") // 去掉 SPA 前缀
		lookupPath := strings.TrimPrefix(reqPath, assetsPrefix)
		lookupPath = strings.TrimPrefix(lookupPath, "/") // 保证无多余 /

		logger.Debug("Assets request path", slog.String("reqPath", reqPath))
		logger.Debug("Assets lookup path", slog.String("lookupPath", lookupPath))

		// ---- 查找文件 ----
		result, err := f.Lookup(finder.NewLookupContext(c.Get(fiber.HeaderAcceptEncoding), lookupPath))
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
		logger.Debug("Finder Result", slog.Any("result", result))
		// ---- 设置响应头 ----
		c.Set(fiber.HeaderContentType, result.MediaTypeString()) // 原始 MIME
		// 如果请求了 Accept-Encoding 但是没有找到压缩文件
		if result.Encoding == "" && c.Get(fiber.HeaderAcceptEncoding) != "" {
			encodings := result.Encoding
			if len(encodings) > 0 {
				options := goeventbus.NewMessageHeadersBuilder().SetHeader("header", "value").Build()
				message := goeventbus.NewMessageBuilder().SetPayload(map[string]interface{}{
					"sourceKey": result.Key,
					"formats":   encodings,
				}).SetHeaders(options).Build()
				dep.EventBus.Channel(constant.CompressEvent).Publisher().Publish(message)
				logger.Debug("Triggered compress event", slog.String("sourceKey", result.Key), slog.Any("formats", encodings))
			}
		}

		// 如果返回了压缩文件，设置 Content-Encoding
		if result.Encoding != "" {
			c.Set(fiber.HeaderContentEncoding, result.Encoding)
		}
		// ---- 日志 ----
		logger.Debug("Serving asset",
			slog.String("path", lookupPath),
			slog.String("mediaType", result.MediaTypeString()),
			slog.String("encoding", result.Encoding),
			slog.Int("size", len(result.Data)),
		)

		// ---- 返回内容 ----
		return c.Send(result.Data)
	})
}
