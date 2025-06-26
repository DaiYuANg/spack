package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
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
	app, cfg, log, total := dep.App, dep.Config, dep.Log, dep.HttpRequestsTotal

	servePath := strings.TrimSpace(cfg.Spa.Path) + "*"
	app.Use(servePath, func(c fiber.Ctx) error {
		incr := func(label string) {
			total.WithLabelValues(c.Method(), c.Path(), label).Inc()
		}
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Spa.Static, reqPath)
		if ok, err := tryServeCompressed(c, fullPath, log); ok && err == nil {
			incr(constant.Compress)
			return nil
		}

		// 普通静态文件
		if ok, err := tryServeStatic(c, fullPath, log); ok && err == nil {
			incr(constant.Normal)
			return nil
		}

		// fallback
		if ok, err := tryServeFallback(c, cfg, log); ok && err == nil {
			incr(constant.Fallback)
			return nil
		}

		incr("not_found")
		return fiber.ErrNotFound
	})
}

func tryServeCompressed(c fiber.Ctx, fullPath string, log *zap.SugaredLogger) (bool, error) {
	acceptEncoding := c.Get(fiber.HeaderAcceptEncoding)
	encodings := lo.Map(strings.Split(acceptEncoding, ","), func(e string, _ int) string {
		return strings.TrimSpace(e)
	})

	enc, found := lo.Find(encodings, func(enc string) bool {
		ext, supported := SupportCompressExt[enc]
		if !supported {
			return false
		}
		compressedPath := fullPath + ext
		return pkg.FileExists(compressedPath)
	})
	if !found {
		return false, nil
	}

	// enc 是支持且存在的编码
	ext := SupportCompressExt[enc]
	compressedPath := fullPath + ext
	content, err := os.ReadFile(compressedPath)
	if err != nil {
		return false, err
	}

	cacheControl := "public, max-age=31536000, immutable"
	contentEncoding := enc
	log.Debugf("Serving compressed file: %s", compressedPath)

	c.Set(fiber.HeaderContentEncoding, contentEncoding)
	c.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
	c.Set(fiber.HeaderCacheControl, cacheControl)
	c.Type(filepath.Ext(fullPath))
	if err := c.Send(content); err != nil {
		return false, err
	}

	return true, nil
}

func tryServeStatic(c fiber.Ctx, fullPath string, log *zap.SugaredLogger) (bool, error) {
	if !pkg.FileExists(fullPath) {
		return false, nil
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return false, err
	}

	ext := filepath.Ext(fullPath)
	cacheControl := lo.Ternary(ext != constant.HTML, "public, max-age=31536000, immutable", "no-cache")

	log.Debugf("Serving static file: %s", fullPath)

	c.Type(ext)
	c.Set(fiber.HeaderCacheControl, cacheControl)
	if err := c.Send(content); err != nil {
		return false, err
	}

	return true, nil
}

func tryServeFallback(c fiber.Ctx, cfg *config.Config, log *zap.SugaredLogger) (bool, error) {
	fallbackPath := filepath.Join(cfg.Spa.Static, cfg.Spa.Fallback)
	if !pkg.FileExists(fallbackPath) {
		return false, nil
	}

	log.Debug("into fallback")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	err := c.SendFile(fallbackPath)

	return err == nil, err
}
