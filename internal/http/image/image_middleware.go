package image

import (
	"bytes"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sproxy/internal/config"
	"strings"
)

var supportedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

type optimizedImageMiddlewareDependency struct {
	fx.In
	Config *config.Config
	Log    *zap.SugaredLogger
}

func optimizedImageMiddleware(config *config.Config, log *zap.SugaredLogger) fiber.Handler {
	return func(c fiber.Ctx) error {

		if config.Spa.Image.Webp == false {
			return c.Next()
		}

		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(config.Spa.Static, reqPath)

		// 路径是否存在
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			return c.Next()
		}

		// 仅处理支持的图片扩展名
		ext := strings.ToLower(filepath.Ext(fullPath))
		if !supportedExts[ext] {
			return c.Next()
		}

		// 判断客户端是否支持 WebP
		accept := c.Get("Accept")
		clientWantsWebP := strings.Contains(accept, "image/webp")

		// 加载原图
		img, err := imaging.Open(fullPath)
		if err != nil {
			log.Warnf("Failed to open image: %v", err)
			return c.Next()
		}

		var buf bytes.Buffer
		if clientWantsWebP {
			log.Infof("Converting to WebP: %s", fullPath)
			// 默认质量 75
			err = webp.Encode(&buf, img, &webp.Options{Quality: 75})
			if err != nil {
				log.Warnf("failed to encode webp: %v", err)
				return c.Next()
			}
			c.Set("Content-Type", "image/webp")
		} else {
			log.Infof("Compressing image: %s", fullPath)
			switch ext {
			case ".jpg", ".jpeg":
				err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(75))
				c.Set("Content-Type", "image/jpeg")
			case ".png":
				err = imaging.Encode(&buf, img, imaging.PNG)
				c.Set("Content-Type", "image/png")
			}
			if err != nil {
				log.Warnf("failed to encode fallback image: %v", err)
				return c.Next()
			}
		}

		c.Set("Cache-Control", "public, max-age=86400")
		return c.SendStream(&buf)
	}
}
