package http

import (
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"path/filepath"
	internal_cache "sproxy/internal/cache"
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
	Cache  *cache.Cache[*internal_cache.CachedFile] `name:"fileCache"`
}

func spaMiddleware(dep SpaMiddlewareDependency) {
	app, cfg, log, _ := dep.App, dep.Config, dep.Log, dep.Cache
	app.Use("/", func(c fiber.Ctx) error {
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Spa.Static, reqPath)
		log.Debugf("fullPath %s", fullPath)
		if pkg.FileExists(fullPath) {
			if filepath.Ext(fullPath) != constant.HTML {
				c.Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Set("Cache-Control", "no-cache")
			}
			return c.SendFile(fullPath)
		}

		// fallback：SPA 场景
		fallbackPath := filepath.Join(cfg.Spa.Static, cfg.Spa.Fallback)
		if pkg.FileExists(fallbackPath) {
			log.Debug("into fallback")
			c.Set("Cache-Control", "no-cache")
			return c.SendFile(fallbackPath)
		}

		// fallback 文件也不存在 → 返回 404
		//return c.Status(404).SendString("404 Not Found")
		return fiber.ErrNotFound
	})
}
