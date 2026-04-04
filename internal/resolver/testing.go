package resolver

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

// NewResolverForTest exposes resolver construction for external tests.
func NewResolverForTest(cfg *config.Assets, cat catalog.Catalog, logger *slog.Logger) *Resolver {
	return newResolverFromDeps(cfg, cat, logger)
}

// ParseAcceptEncodingForTest exposes encoding preference parsing for external tests.
func ParseAcceptEncodingForTest(header string) collectionx.List[string] {
	return parseAcceptEncoding(header)
}

// ParseAcceptImageFormatsForTest exposes image format preference parsing for external tests.
func ParseAcceptImageFormatsForTest(header, sourceFormat string) collectionx.List[string] {
	return parseAcceptImageFormats(header, sourceFormat)
}
