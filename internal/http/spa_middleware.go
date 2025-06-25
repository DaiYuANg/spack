package http

import (
	"context"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sproxy/internal/config"
	"sproxy/internal/constant"
	"sproxy/pkg"
	"strings"
)

type SpaMiddlewareDependency struct {
	fx.In
	App               *fiber.App
	Config            *config.Config
	Log               *zap.SugaredLogger
	HttpRequestsTotal *prometheus.CounterVec
	Cache             *cache.Cache[[]byte]
}

var SupportCompressExt = map[string]string{
	"br":   ".br",
	"gzip": ".gz",
	"svgz": ".svgz",
	"zz":   ".zz",
	"xz":   ".xz",
	"lz":   ".lz",
	"lz4":  ".lz4",
	"zst":  ".zst",
}

func spaMiddleware(dep SpaMiddlewareDependency) {
	app, cfg, log, total, fileCache := dep.App, dep.Config, dep.Log, dep.HttpRequestsTotal, dep.Cache

	app.Use("/*", func(c fiber.Ctx) error {
		incr := func(label string) {
			total.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Spa.Static, reqPath)
		cacheKey := pkg.HashKey(fullPath)
		log.Debugf("Hash key: %s", cacheKey)
		// 先尝试缓存
		//data, err := fileCache.Get(context.Background(), cacheKey)
		//if err != nil {
		//  log.Warnw("cache load error", "err", err)
		//}
		//if err == nil && data != nil {
		//  c.Type(filepath.Ext(fullPath))
		//  c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		//  log.Debug("load file from cache")
		//  return c.Send(data)
		//}
		//
		if ok, content, err := tryServeCompressed(c, fullPath, log); ok && err == nil {
			incr("compress")
			err = fileCache.Set(context.Background(), cacheKey, content)
			if err != nil {
				log.Warnw("cache set error", "err", err)
			}
			return nil
		}

		// 普通静态文件
		if ok, content, err := tryServeStatic(c, fullPath, log); ok && err == nil {
			incr("normal")
			err = fileCache.Set(context.Background(), cacheKey, content)
			if err != nil {
				log.Warnw("cache set error", "err", err)
			}
			return nil
		}

		// fallback
		if tryServeFallback(c, cfg, log) {
			incr("fallback")
			return nil
		}

		incr("not_found")
		return fiber.ErrNotFound
	})
}

func tryServeCompressed(c fiber.Ctx, fullPath string, log *zap.SugaredLogger) (bool, []byte, error) {
	acceptEncoding := c.Get(fiber.HeaderAcceptEncoding)
	for _, e := range strings.Split(acceptEncoding, ",") {
		enc := strings.TrimSpace(e)
		ext, supported := SupportCompressExt[enc]
		if !supported {
			continue
		}
		compressedPath := fullPath + ext

		if !pkg.FileExists(compressedPath) {
			continue
		}

		content, err := os.ReadFile(compressedPath)
		if err != nil {
			return false, nil, err
		}

		c.Set(fiber.HeaderContentEncoding, enc)
		c.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		c.Type(filepath.Ext(fullPath))

		if err := c.Send(content); err != nil {
			return false, nil, err
		}

		log.Debugf("Serving compressed file: %s", compressedPath)
		return true, content, nil
	}

	return false, nil, nil
}

func tryServeStatic(c fiber.Ctx, fullPath string, log *zap.SugaredLogger) (bool, []byte, error) {
	if !pkg.FileExists(fullPath) {
		return false, nil, nil
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return false, nil, err
	}

	if filepath.Ext(fullPath) != constant.HTML {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	} else {
		c.Set(fiber.HeaderCacheControl, "no-cache")
	}
	c.Type(filepath.Ext(fullPath))

	if err := c.Send(content); err != nil {
		return false, nil, err
	}

	log.Debugf("Serving static file: %s", fullPath)
	return true, content, nil
}

func tryServeFallback(c fiber.Ctx, cfg *config.Config, log *zap.SugaredLogger) bool {
	fallbackPath := filepath.Join(cfg.Spa.Static, cfg.Spa.Fallback)
	if pkg.FileExists(fallbackPath) {
		log.Debug("into fallback")
		c.Set("Cache-Control", "no-cache")
		_ = c.SendFile(fallbackPath)
		return true
	}
	return false
}
