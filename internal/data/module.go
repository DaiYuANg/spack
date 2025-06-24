package data

import (
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

var Module = fx.Module("data",
	fx.Provide(
		newBbolt,
	),
)

type NewBblotDependency struct {
	fx.In
	Logger  *zap.SugaredLogger
	BaseDir string `name:"baseDir"`
}

func newBbolt(dep NewBblotDependency) (*bbolt.DB, error) {
	baseDir, log := dep.BaseDir, dep.Logger
	_, err := os.Stat(baseDir)
	if err != nil {
		log.Debugf("base dir %s does not exist", baseDir)
		err := os.MkdirAll(baseDir, 0755)
		if err != nil {
			log.Fatalf("error creating base dir %s: %v", baseDir, err)
			return nil, err
		}
	}
	dbPath := filepath.Join(baseDir, "sproxy.db")
	log.Debugf("db path %s", dbPath)
	return bbolt.Open(dbPath, 0600, nil)
}
