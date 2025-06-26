package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"path/filepath"
	"sproxy/pkg"
	"strings"
)

func spaCompressMiddleware(log *zap.SugaredLogger) fiber.Handler {
	return func(c fiber.Ctx) error {
		fullPath := c.Locals("spa:fullPath").(string)
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
			return c.Next()
		}
		incr, ok := c.Locals("spa:incr").(func(string))
		if !ok {
			return c.Next() // 没有打点函数则跳过
		}
		// enc 是支持且存在的编码
		ext := SupportCompressExt[enc]
		compressedPath := fullPath + ext

		cacheControl := "public, max-age=31536000, immutable"
		contentEncoding := enc
		log.Debugf("Serving compressed file: %s", compressedPath)

		c.Set(fiber.HeaderContentEncoding, contentEncoding)
		c.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
		c.Set(fiber.HeaderCacheControl, cacheControl)
		c.Type(filepath.Ext(fullPath))
		incr("compress")
		return c.SendFile(compressedPath)
	}
}
