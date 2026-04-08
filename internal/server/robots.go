package server

import (
	"embed"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

const robotsAssetPath = "robots.txt"

//go:embed templates/robots.txt.tmpl
var robotsTemplateFS embed.FS

var robotsContentTemplate = template.Must(template.ParseFS(robotsTemplateFS, "templates/robots.txt.tmpl"))

type robotsTemplateData struct {
	UserAgent string
	Allow     string
	Disallow  string
	Host      string
	Sitemap   string
}

func registerRobotsRoute(
	app *fiber.App,
	cfg *config.Config,
	logger *slog.Logger,
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
) {
	if !cfg.Robots.Enable {
		return
	}

	handler := func(c fiber.Ctx) error {
		if asset, ok := staticRobotsAsset(cfg.Robots, cat); ok {
			_, err := sendResolvedAsset(
				c,
				cfg,
				resolver.Request{RangeRequested: strings.TrimSpace(c.Get(fiber.HeaderRange)) != ""},
				&resolver.Result{
					Asset:     asset,
					FilePath:  asset.FullPath,
					MediaType: asset.MediaType,
					ETag:      asset.ETag,
				},
				"",
				logger,
				bodyCache,
			)
			return err
		}
		return sendGeneratedRobots(c, cfg.Robots)
	}

	app.Get("/robots.txt", handler)
	app.Head("/robots.txt", handler)
}

func staticRobotsAsset(cfg config.Robots, cat catalog.Catalog) (*catalog.Asset, bool) {
	if cfg.Override {
		return nil, false
	}
	return cat.FindAsset(robotsAssetPath)
}

func sendGeneratedRobots(c fiber.Ctx, cfg config.Robots) error {
	body, err := renderRobotsContent(cfg)
	if err != nil {
		return fmt.Errorf("render robots.txt: %w", err)
	}

	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	c.Status(fiber.StatusOK)
	if c.Method() == fiber.MethodHead {
		return nil
	}
	if err := c.SendString(body); err != nil {
		return fmt.Errorf("send generated robots.txt: %w", err)
	}
	return nil
}

func renderRobotsContent(cfg config.Robots) (string, error) {
	allow := strings.TrimSpace(cfg.Allow)
	disallow := strings.TrimSpace(cfg.Disallow)
	if allow == "" && disallow == "" {
		allow = "/"
	}

	var body strings.Builder
	err := robotsContentTemplate.Execute(&body, robotsTemplateData{
		UserAgent: cfg.NormalizedUserAgent(),
		Allow:     allow,
		Disallow:  disallow,
		Host:      strings.TrimSpace(cfg.Host),
		Sitemap:   strings.TrimSpace(cfg.Sitemap),
	})
	if err != nil {
		return "", fmt.Errorf("execute robots.txt template: %w", err)
	}

	return strings.TrimRight(body.String(), "\n") + "\n", nil
}
