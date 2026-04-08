package resolver

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
	contentcodingspec "github.com/daiyuang/spack/internal/contentcoding/spec"
)

// NewResolverForTest exposes resolver construction for external tests.
func NewResolverForTest(cfg *config.Assets, cat catalog.Catalog, logger *slog.Logger) *Resolver {
	defaults := config.DefaultConfig().Compression
	return newResolver(cfg, contentcoding.NewRegistry(contentcoding.Options{
		BrotliQuality: defaults.BrotliQuality,
		GzipLevel:     defaults.GzipLevel,
		ZstdLevel:     defaults.ZstdLevel,
	}, defaults.NormalizedEncodings()), cat, logger)
}

// NewResolverWithCompressionForTest exposes resolver construction with compression config for external tests.
func NewResolverWithCompressionForTest(
	cfg *config.Assets,
	compression *config.Compression,
	cat catalog.Catalog,
	logger *slog.Logger,
) *Resolver {
	return newResolver(cfg, contentcoding.NewRegistry(contentcoding.Options{
		BrotliQuality: compression.BrotliQuality,
		GzipLevel:     compression.GzipLevel,
		ZstdLevel:     compression.ZstdLevel,
	}, compression.NormalizedEncodings()), cat, logger)
}

// ParseAcceptEncodingForTest exposes encoding preference parsing for external tests.
func ParseAcceptEncodingForTest(header string) collectionx.List[string] {
	return parseAcceptEncoding(header, contentcodingspec.DefaultNames())
}

// ParseAcceptEncodingWithSupportedForTest exposes encoding preference parsing with a custom support list for external tests.
func ParseAcceptEncodingWithSupportedForTest(header string, supported collectionx.List[string]) collectionx.List[string] {
	return parseAcceptEncoding(header, supported)
}

// ParseAcceptImageFormatsForTest exposes image format preference parsing for external tests.
func ParseAcceptImageFormatsForTest(header, sourceFormat string) collectionx.List[string] {
	return parseAcceptImageFormats(header, sourceFormat)
}
