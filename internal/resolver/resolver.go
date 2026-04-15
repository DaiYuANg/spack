package resolver

import (
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
	contentcodingspec "github.com/daiyuang/spack/internal/contentcoding/spec"
	"github.com/daiyuang/spack/internal/media"
	"github.com/daiyuang/spack/internal/requestpath"
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
	if obs != nil {
		obs = observabilityx.Normalize(obs, logger)
	}
	return &Resolver{
		cfg:                cfg,
		supportedEncodings: contentcodingspec.NormalizeNames(registry.Names()),
		catalog:            cat,
		logger:             logger,
		obs:                obs,
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
	if request.Width > 0 || listLen(preferredImageFormats) > 0 {
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

	if !request.RangeRequested && listLen(encodings) > 0 {
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
			go r.recordMetrics(startedAt, result, nil)
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
	go r.recordMetrics(startedAt, result, nil)
	return result, nil
}

func (r *Resolver) ResolveAfterVariantArtifactMiss(request Request, variant *catalog.Variant) (*Result, error) {
	if r == nil {
		return nil, ErrNotFound
	}
	if variant != nil {
		r.catalog.DeleteVariantByArtifactPath(variant.ArtifactPath)
	}
	return r.Resolve(request)
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

	if count := int64(listLen(result.PreferredEncodings)); count > 0 {
		r.obs.Counter(resolverGenerationRequestsTotalSpec).Add(context.Background(), count,
			observabilityx.String("kind", "encoding"),
		)
	}
	if count := int64(listLen(result.PreferredWidths)); count > 0 {
		r.obs.Counter(resolverGenerationRequestsTotalSpec).Add(context.Background(), count,
			observabilityx.String("kind", "image_width"),
		)
	}
	if count := int64(listLen(result.PreferredFormats)); count > 0 {
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
	resolvedPath := requestpath.Clean(requestPath)
	if asset, ok := r.findPrimaryAsset(resolvedPath); ok {
		return asset, false
	}

	if r.cfg.Fallback.On == config.FallbackOnNotFound && resolvedPath.AllowsEntryFallback {
		target := requestpath.Clean(r.cfg.Fallback.Target).Value
		if target != "" {
			if asset, ok := findAssetForRead(r.catalog, target); ok {
				return asset, true
			}
		}
	}
	return nil, false
}

func (r *Resolver) findPrimaryAsset(requestPath requestpath.Cleaned) (*catalog.Asset, bool) {
	if requestPath.Value == "" {
		return findAssetForRead(r.catalog, r.cfg.Entry)
	}

	if asset, ok := findAssetForRead(r.catalog, requestPath.Value); ok {
		return asset, true
	}
	if !requestPath.AllowsEntryFallback {
		return nil, false
	}

	candidate := path.Join(requestPath.Value, r.cfg.Entry)
	if candidate == requestPath.Value {
		return nil, false
	}
	return findAssetForRead(r.catalog, candidate)
}

func (r *Resolver) pickVariant(asset *catalog.Asset, encodings collectionx.List[string]) *catalog.Variant {
	variants := listVariantsForRead(r.catalog, asset.Path)
	usable := newVariantUsabilityCache()

	var picked *catalog.Variant
	encodings.Range(func(_ int, encoding string) bool {
		variants.Range(func(_ int, variant *catalog.Variant) bool {
			if variant.Encoding != encoding || !usable.IsUsable(variant, asset.SourceHash) {
				return true
			}
			picked = variant
			return false
		})
		return picked == nil
	})
	return picked
}

func (r *Resolver) pickImageVariant(asset *catalog.Asset, width int, formats collectionx.List[string]) *catalog.Variant {
	sourceFormat := media.ImageFormat(asset.MediaType)
	variants := listVariantsForRead(r.catalog, asset.Path)
	if variants.IsEmpty() {
		return nil
	}

	if formats.IsEmpty() {
		formats = collectionx.NewList(sourceFormat)
	}

	usable := newVariantUsabilityCache()
	var picked *catalog.Variant
	formats.Range(func(_ int, format string) bool {
		picked = pickImageVariantForFormat(variants, usable, asset.SourceHash, format, sourceFormat, width)
		return picked == nil
	})
	return picked
}
