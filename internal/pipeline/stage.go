package pipeline

import "github.com/daiyuang/spack/internal/catalog"

type Request struct {
	AssetPath          string
	PreferredEncodings []string
	PreferredFormats   []string
	PreferredWidths    []int
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
