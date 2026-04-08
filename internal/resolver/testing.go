package resolver

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
)

// NewResolverForTest exposes resolver construction for external tests.
func NewResolverForTest(cfg *config.Assets, cat catalog.Catalog, logger *slog.Logger) *Resolver {
	compression := config.DefaultConfig().Compression
	return newResolverFromDeps(cfg, &compression, cat, logger)
}

// NewResolverWithCompressionForTest exposes resolver construction with compression config for external tests.
func NewResolverWithCompressionForTest(
	cfg *config.Assets,
	compression *config.Compression,
	cat catalog.Catalog,
	logger *slog.Logger,
) *Resolver {
	return newResolverFromDeps(cfg, compression, cat, logger)
}

// ParseAcceptEncodingForTest exposes encoding preference parsing for external tests.
func ParseAcceptEncodingForTest(header string) collectionx.List[string] {
	return parseAcceptEncoding(header, contentcoding.DefaultNames())
}

// ParseAcceptEncodingWithSupportedForTest exposes encoding preference parsing with a custom support list for external tests.
func ParseAcceptEncodingWithSupportedForTest(header string, supported collectionx.List[string]) collectionx.List[string] {
	return parseAcceptEncoding(header, supported)
}

// ParseAcceptImageFormatsForTest exposes image format preference parsing for external tests.
func ParseAcceptImageFormatsForTest(header, sourceFormat string) collectionx.List[string] {
	return parseAcceptImageFormats(header, sourceFormat)
}
