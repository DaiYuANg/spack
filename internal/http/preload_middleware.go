package http

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

func parsePreloadLinksFromHTML(htmlPath string, logger *zap.SugaredLogger) (map[string]string, error) {
	file, err := os.Open(htmlPath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		lo.Must0(file.Close(), err.Error())
	}(file)

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		return nil, err
	}

	links := make(map[string]string)

	// 提取 script 标签
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src != "" {
			links[src] = "script"
		}
	})

	// 提取 stylesheet
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href != "" {
			links[href] = "style"
		}
	})

	// 提取已有的 preload 标签
	doc.Find("link[rel='preload']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		as, _ := s.Attr("as")
		if href != "" && as != "" {
			links[href] = as
		}
	})

	logger.Debugf("preload links: %v", links)

	return links, nil
}

func preloadMiddleware(preloadMap map[string]string, logger *zap.SugaredLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		// 只针对 HTML 响应添加 Link
		if strings.HasPrefix(c.Get("Content-Type"), "text/html") {
			logger.Debugf("into preload%s", c.OriginalURL())
			var headers []string
			for href, asType := range preloadMap {
				// 可加更多 type 判断，如字体
				if asType == "font" {
					headers = append(headers, fmt.Sprintf("<%s>; rel=preload; as=font; type=\"font/woff2\"; crossorigin", href))
				} else {
					headers = append(headers, fmt.Sprintf("<%s>; rel=preload; as=%s", href, asType))
				}
			}
			c.Set("Link", strings.Join(headers, ", "))
		}

		return err
	}
}

func setupPreload(app *fiber.App, config *config.Config, logger *zap.SugaredLogger) {
	if !config.Spa.Preload {
		return
	}
	htmlPath := filepath.Join(config.Spa.Static, "index.html")
	preloads, err := parsePreloadLinksFromHTML(htmlPath, logger)
	if err != nil {
		logger.Fatalf("Failed to parse preload links: %v", err)
	}
	app.Use(preloadMiddleware(preloads, logger))
}
