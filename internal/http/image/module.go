package image

import (
	"github.com/gofiber/fiber/v3"
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sproxy/internal/config"
)

var Module = fx.Module("image",
	fx.Provide(newKvMap),
	fx.Invoke(
		prepareScan,
		scan,
		imageMiddleware,
	),
)

func newKvMap() (*bbolt.DB, error) {
	tempDir := filepath.Join(os.TempDir(), "sproxy")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(tempDir, "sproxy.image.db")
	return bbolt.Open(dbPath, 0600, nil)
}

type MiddlewareDependency struct {
	fx.In
	App    *fiber.App
	Config *config.Config
	Log    *zap.SugaredLogger
	KvMap  *bbolt.DB
}

func imageMiddleware(dep MiddlewareDependency) {
	app, cfg, log := dep.App, dep.Config, dep.Log
	app.Use(optimizedImageMiddleware(cfg, log))
}
