package image

import (
	"bytes"
	"fmt"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"image"
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
		if !config.Spa.Image.Webp {
			return c.Next()
		}

		reqPath := strings.TrimPrefix(c.Path(), "/")
		fullPath := filepath.Join(config.Spa.Static, reqPath)

		if skipFile(fullPath) {
			return c.Next()
		}

		ext := strings.ToLower(filepath.Ext(fullPath))
		if !supportedExts[ext] {
			return c.Next()
		}

		img, err := loadImage(fullPath, log)
		if err != nil {
			return c.Next()
		}

		isWebP := clientAcceptsWebP(c)

		buf, contentType, err := encodeImage(img, ext, isWebP, log)
		if err != nil {
			return c.Next()
		}

		c.Set("Content-Type", contentType)
		c.Set("Cache-Control", "public, max-age=86400")
		return c.SendStream(bytes.NewReader(buf))
	}
}

func skipFile(path string) bool {
	info, err := os.Stat(path)
	return err != nil || info.IsDir()
}

func clientAcceptsWebP(c fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderAccept), "image/webp")
}

func loadImage(path string, log *zap.SugaredLogger) (image.Image, error) {
	img, err := imaging.Open(path)
	if err != nil {
		log.Warnf("Failed to open image: %v", err)
		return nil, err
	}
	return img, nil
}

func encodeImage(img image.Image, ext string, toWebP bool, log *zap.SugaredLogger) ([]byte, string, error) {
	var buf bytes.Buffer
	if toWebP {
		log.Infof("Converting to WebP")
		if err := webp.Encode(&buf, img, &webp.Options{Quality: 75}); err != nil {
			log.Warnf("WebP encode failed: %v", err)
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil
	}

	log.Infof("Compressing original image")
	switch ext {
	case ".jpg", ".jpeg":
		err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(75))
		return buf.Bytes(), "image/jpeg", err
	case ".png":
		err := imaging.Encode(&buf, img, imaging.PNG)
		return buf.Bytes(), "image/png", err
	default:
		return nil, "", fmt.Errorf("unsupported image format")
	}
}
