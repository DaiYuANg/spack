package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"path/filepath"
	"sproxy/internal/config"
	"sproxy/internal/constant"
	"sproxy/pkg"
	"strings"
)

type SpaMiddlewareDependency struct {
	fx.In
	App    *fiber.App
	Config *config.Config
	Log    *zap.SugaredLogger
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
	app, cfg, log := dep.App, dep.Config, dep.Log

	app.Use("/*", func(c fiber.Ctx) error {
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Spa.Static, reqPath)

		// 优先尝试压缩文件
		if tryServeCompressed(c, fullPath, log) {
			return nil
		}

		// 普通静态文件
		if tryServeStatic(c, fullPath, log) {
			return nil
		}

		// fallback
		if tryServeFallback(c, cfg, log) {
			return nil
		}

		return fiber.ErrNotFound
	})
}

func tryServeCompressed(c fiber.Ctx, fullPath string, log *zap.SugaredLogger) bool {
	acceptEncoding := c.Get(fiber.HeaderAcceptEncoding)

	// 查找第一个存在的压缩版本
	if encoding, ok := lo.Find(strings.Split(acceptEncoding, ","), func(e string) bool {
		enc := strings.TrimSpace(e)
		ext, supported := SupportCompressExt[enc]
		if !supported {
			return false
		}
		compressedPath := fullPath + ext
		return pkg.FileExists(compressedPath)
	}); ok {
		enc := strings.TrimSpace(encoding)
		ext := SupportCompressExt[enc]
		compressedPath := fullPath + ext

		log.Debugf("Serving compressed file: %s", compressedPath)

		c.Set(fiber.HeaderContentEncoding, enc)
		c.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		c.Type(filepath.Ext(fullPath))
		_ = c.SendFile(compressedPath)
		return true
	}

	return false
}

func tryServeStatic(c fiber.Ctx, fullPath string, log *zap.SugaredLogger) bool {
	if !pkg.FileExists(fullPath) {
		return false
	}

	if filepath.Ext(fullPath) != constant.HTML {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	} else {
		c.Set(fiber.HeaderCacheControl, "no-cache")
	}
	_ = c.SendFile(fullPath)
	log.Debugf("Serving static file: %s", fullPath)
	return true
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
