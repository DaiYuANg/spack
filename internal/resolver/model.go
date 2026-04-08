package resolver

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

type Request struct {
	Path           string
	Accept         string
	AcceptEncoding string
	Width          int
	Format         string
	RangeRequested bool
}

type Result struct {
	Asset              *catalog.Asset
	Variant            *catalog.Variant
	FilePath           string
	MediaType          string
	ContentEncoding    string
	ETag               string
	PreferredEncodings collectionx.List[string]
	PreferredWidths    collectionx.List[int]
	PreferredFormats   collectionx.List[string]
	FallbackUsed       bool
}

type Resolver struct {
	cfg                *config.Assets
	supportedEncodings collectionx.List[string]
	catalog            catalog.Catalog
	logger             *slog.Logger
	obs                observabilityx.Observability
}
