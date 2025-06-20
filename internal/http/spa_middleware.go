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
	app, cfg, log, _ := dep.App, dep.Config, dep.Log, dep.Cache

	app.Use("/", func(c fiber.Ctx) error {
		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(cfg.Spa.Static, reqPath)

		acceptEncoding := c.Get("Accept-Encoding")
		encodingList := strings.Split(acceptEncoding, ",")
		log.Debugf("Accept-Encoding: %s", acceptEncoding)
		// 遍历支持的压缩格式
		for _, enc := range encodingList {
			enc = strings.TrimSpace(enc)
			if ext, ok := SupportCompressExt[enc]; ok {
				compressedPath := fullPath + ext
				if pkg.FileExists(compressedPath) {
					log.Debugf("Serving compressed file: %s", compressedPath)

					c.Set("Content-Encoding", enc)
					c.Set("Vary", "Accept-Encoding")
					c.Set("Cache-Control", "public, max-age=31536000, immutable")

					c.Type(filepath.Ext(fullPath))
					log.Debugf("compress file%s", fullPath)
					return c.SendFile(compressedPath)
				}
			}
		}

		// 没有压缩版本，回退到普通文件
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
		return fiber.ErrNotFound
	})
}
