package pipeline

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
)

type Request struct {
	AssetPath          string
	PreferredEncodings collectionx.List[string]
	PreferredFormats   collectionx.List[string]
	PreferredWidths    collectionx.List[int]
}

type Task struct {
	AssetPath string
	Encoding  string
	Format    string
	Width     int
}

type Stage interface {
	Name() string
	Plan(asset *catalog.Asset, request Request) []Task
	Execute(task Task, asset *catalog.Asset) (*catalog.Variant, error)
}
