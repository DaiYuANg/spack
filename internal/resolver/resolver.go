package resolver

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/media"
)

var (
	ErrNotFound = errors.New("asset not found")

	resolverResolutionsTotalSpec = observabilityx.NewCounterSpec(
		"resolver_resolutions_total",
		observabilityx.WithDescription("Total number of asset resolution attempts."),
		observabilityx.WithLabelKeys("result"),
	)
	resolverResolutionDurationSpec = observabilityx.NewHistogramSpec(
		"resolver_resolution_duration_seconds",
		observabilityx.WithDescription("Asset resolution duration in seconds."),
		observabilityx.WithUnit("s"),
		observabilityx.WithLabelKeys("result"),
	)
	resolverGenerationRequestsTotalSpec = observabilityx.NewCounterSpec(
		"resolver_generation_requests_total",
		observabilityx.WithDescription("Total number of requested generated artifact dimensions by kind."),
		observabilityx.WithLabelKeys("kind"),
	)
)

func newResolver(
	cfg *config.Assets,
	registry contentcoding.Registry,
	cat catalog.Catalog,
	logger *slog.Logger,
	obs observabilityx.Observability,
) *Resolver {
	return &Resolver{
		cfg:                cfg,
		supportedEncodings: registry.Names(),
		catalog:            cat,
		logger:             logger,
		obs:                observabilityx.Normalize(obs, logger),
	}
}

func (r *Resolver) Resolve(request Request) (*Result, error) {
	startedAt := time.Now()
	asset, fallbackUsed := r.findAsset(request.Path)
	if asset == nil {
		r.recordMetrics(startedAt, nil, ErrNotFound)
		return nil, ErrNotFound
	}

	encodings := parseAcceptEncoding(request.AcceptEncoding, r.supportedEncodings)
	requestedFormat := media.NormalizeImageFormat(request.Format)
	preferredImageFormats := preferredImageFormats(request.Accept, requestedFormat, asset.MediaType)
	if request.Width > 0 || preferredImageFormats.Len() > 0 {
		if variant := r.pickImageVariant(asset, request.Width, preferredImageFormats); variant != nil {
			result := &Result{
				Asset:        asset,
				Variant:      variant,
				FilePath:     variant.ArtifactPath,
				MediaType:    firstNonEmpty(variant.MediaType, asset.MediaType),
				ETag:         firstNonEmpty(variant.ETag, asset.ETag),
				FallbackUsed: fallbackUsed,
			}
			r.recordMetrics(startedAt, result, nil)
			return result, nil
		}
	}

	if !request.RangeRequested && encodings.Len() > 0 {
		if variant := r.pickVariant(asset, encodings); variant != nil {
			result := &Result{
				Asset:           asset,
				Variant:         variant,
				FilePath:        variant.ArtifactPath,
				MediaType:       asset.MediaType,
				ContentEncoding: variant.Encoding,
				ETag:            firstNonEmpty(variant.ETag, asset.ETag),
				FallbackUsed:    fallbackUsed,
			}
			r.recordMetrics(startedAt, result, nil)
			return result, nil
		}
	}

	result := &Result{
		Asset:              asset,
		FilePath:           asset.FullPath,
		MediaType:          asset.MediaType,
		ETag:               asset.ETag,
		PreferredEncodings: encodings,
		PreferredWidths:    preferredWidths(request.Width),
		PreferredFormats:   preferredImageFormats,
		FallbackUsed:       fallbackUsed,
	}
	r.recordMetrics(startedAt, result, nil)
	return result, nil
}

func (r *Resolver) recordMetrics(startedAt time.Time, result *Result, err error) {
	if r == nil || r.obs == nil {
		return
	}

	attrs := []observabilityx.Attribute{
		observabilityx.String("result", resolutionResultKind(result, err)),
	}
	r.obs.Counter(resolverResolutionsTotalSpec).Add(context.Background(), 1, attrs...)
	r.obs.Histogram(resolverResolutionDurationSpec).Record(context.Background(), time.Since(startedAt).Seconds(), attrs...)

	if result == nil {
		return
	}

	if count := int64(result.PreferredEncodings.Len()); count > 0 {
		r.obs.Counter(resolverGenerationRequestsTotalSpec).Add(context.Background(), count,
			observabilityx.String("kind", "encoding"),
		)
	}
	if count := int64(result.PreferredWidths.Len()); count > 0 {
		r.obs.Counter(resolverGenerationRequestsTotalSpec).Add(context.Background(), count,
			observabilityx.String("kind", "image_width"),
		)
	}
	if count := int64(result.PreferredFormats.Len()); count > 0 {
		r.obs.Counter(resolverGenerationRequestsTotalSpec).Add(context.Background(), count,
			observabilityx.String("kind", "image_format"),
		)
	}
}

func resolutionResultKind(result *Result, err error) string {
	if errors.Is(err, ErrNotFound) {
		return "not_found"
	}
	if err != nil {
		return "error"
	}
	if result == nil {
		return "empty"
	}
	if kind, ok := resolutionVariantKind(result.Variant); ok {
		return kind
	}
	if result.FallbackUsed {
		return "fallback_asset"
	}
	return "asset"
}

func resolutionVariantKind(variant *catalog.Variant) (string, bool) {
	if variant == nil {
		return "", false
	}
	if variant.Width > 0 || strings.TrimSpace(variant.Format) != "" {
		return "image_variant", true
	}
	if strings.TrimSpace(variant.Encoding) != "" {
		return "encoding_variant", true
	}
	return "variant", true
}

func (r *Resolver) findAsset(requestPath string) (*catalog.Asset, bool) {
	var asset *catalog.Asset
	candidates(requestPath, r.cfg.Entry).Range(func(_ int, candidate string) bool {
		if found, ok := r.catalog.FindAsset(candidate); ok {
			asset = found
			return false
		}
		return true
	})
	if asset != nil {
		return asset, false
	}

	if r.cfg.Fallback.On == config.FallbackOnNotFound {
		target := normalizeAssetPath(r.cfg.Fallback.Target)
		if target != "" {
			if asset, ok := r.catalog.FindAsset(target); ok {
				return asset, true
			}
		}
	}
	return nil, false
}

func (r *Resolver) pickVariant(asset *catalog.Asset, encodings collectionx.List[string]) *catalog.Variant {
	variants := r.catalog.ListVariants(asset.Path)

	var picked *catalog.Variant
	encodings.Range(func(_ int, encoding string) bool {
		picked, _ = variants.FirstWhere(func(_ int, variant *catalog.Variant) bool {
			return variant.Encoding == encoding && isUsableVariant(variant, asset.SourceHash)
		}).Get()
		return picked == nil
	})
	return picked
}

func (r *Resolver) pickImageVariant(asset *catalog.Asset, width int, formats collectionx.List[string]) *catalog.Variant {
	sourceFormat := media.ImageFormat(asset.MediaType)
	candidates := imageCandidates(r.catalog.ListVariants(asset.Path), asset.SourceHash)
	if candidates.IsEmpty() {
		return nil
	}

	if formats.IsEmpty() {
		formats = collectionx.NewList(sourceFormat)
	}

	var picked *catalog.Variant
	formats.Range(func(_ int, format string) bool {
		picked = pickImageVariantForFormat(candidates, format, sourceFormat, width)
		return picked == nil
	})
	return picked
}

func imageCandidates(variants collectionx.List[*catalog.Variant], sourceHash string) collectionx.List[*catalog.Variant] {
	return variants.Where(func(_ int, variant *catalog.Variant) bool {
		if variant.Width <= 0 && variant.Format == "" {
			return false
		}
		return isUsableVariant(variant, sourceHash)
	})
}

func pickImageVariantForFormat(
	candidates collectionx.List[*catalog.Variant],
	format string,
	sourceFormat string,
	width int,
) *catalog.Variant {
	byFormat := candidates.Where(func(_ int, candidate *catalog.Variant) bool {
		return format == "" || variantFormat(candidate, sourceFormat) == format
	})
	if byFormat.IsEmpty() {
		return nil
	}
	if width <= 0 {
		variant, _ := byFormat.FirstWhere(func(_ int, candidate *catalog.Variant) bool {
			return candidate.Width == 0
		}).Get()
		return variant
	}

	byFormat.Sort(func(left, right *catalog.Variant) int {
		return cmp.Compare(left.Width, right.Width)
	})
	if variant, ok := byFormat.FirstWhere(func(_ int, candidate *catalog.Variant) bool {
		return candidate.Width >= width
	}).Get(); ok {
		return variant
	}

	variant, _ := byFormat.GetLast()
	return variant
}

func candidates(requestPath, entry string) collectionx.List[string] {
	normalized := normalizeAssetPath(requestPath)
	if normalized == "" {
		return collectionx.NewList(entry)
	}

	result := collectionx.NewList(normalized)
	if strings.HasSuffix(strings.TrimSpace(requestPath), "/") || path.Ext(normalized) == "" {
		result.Add(path.Join(normalized, entry))
	}
	return uniqueStrings(result)
}

func normalizeAssetPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	clean := path.Clean("/" + strings.TrimPrefix(trimmed, "/"))
	if clean == "/" || clean == "." {
		return ""
	}

	return strings.TrimPrefix(clean, "/")
}
