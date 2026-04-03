package resolver

import (
	"errors"
	"log/slog"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
)

var ErrNotFound = errors.New("asset not found")

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
	PreferredEncodings []string
	PreferredWidths    []int
	PreferredFormats   []string
	FallbackUsed       bool
}

type Resolver struct {
	cfg     *config.Assets
	catalog catalog.Catalog
	logger  *slog.Logger
}

type resolverIn struct {
	Config  *config.Assets
	Catalog catalog.Catalog
	Logger  *slog.Logger
}

func newResolver(in resolverIn) *Resolver {
	return &Resolver{
		cfg:     in.Config,
		catalog: in.Catalog,
		logger:  in.Logger,
	}
}

func newResolverFromDeps(cfg *config.Assets, cat catalog.Catalog, logger *slog.Logger) *Resolver {
	return newResolver(resolverIn{
		Config:  cfg,
		Catalog: cat,
		Logger:  logger,
	})
}

func (r *Resolver) Resolve(request Request) (*Result, error) {
	asset, fallbackUsed := r.findAsset(request.Path)
	if asset == nil {
		return nil, ErrNotFound
	}

	encodings := parseAcceptEncoding(request.AcceptEncoding)
	requestedFormat := normalizeImageFormat(request.Format)
	preferredImageFormats := preferredImageFormats(request.Accept, requestedFormat, asset.MediaType)
	if request.Width > 0 || len(preferredImageFormats) > 0 {
		if variant := r.pickImageVariant(asset, request.Width, preferredImageFormats); variant != nil {
			return &Result{
				Asset:        asset,
				Variant:      variant,
				FilePath:     variant.ArtifactPath,
				MediaType:    firstNonEmpty(variant.MediaType, asset.MediaType),
				ETag:         firstNonEmpty(variant.ETag, asset.ETag),
				FallbackUsed: fallbackUsed,
			}, nil
		}
	}

	if !request.RangeRequested && len(encodings) > 0 {
		if variant := r.pickVariant(asset, encodings); variant != nil {
			return &Result{
				Asset:           asset,
				Variant:         variant,
				FilePath:        variant.ArtifactPath,
				MediaType:       asset.MediaType,
				ContentEncoding: variant.Encoding,
				ETag:            firstNonEmpty(variant.ETag, asset.ETag),
				FallbackUsed:    fallbackUsed,
			}, nil
		}
	}

	return &Result{
		Asset:              asset,
		FilePath:           asset.FullPath,
		MediaType:          asset.MediaType,
		ETag:               asset.ETag,
		PreferredEncodings: encodings,
		PreferredWidths:    preferredWidths(request.Width),
		PreferredFormats:   preferredImageFormats,
		FallbackUsed:       fallbackUsed,
	}, nil
}

func (r *Resolver) findAsset(requestPath string) (*catalog.Asset, bool) {
	for _, candidate := range candidates(requestPath, r.cfg.Entry) {
		if asset, ok := r.catalog.FindAsset(candidate); ok {
			return asset, false
		}
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

func (r *Resolver) pickVariant(asset *catalog.Asset, encodings []string) *catalog.Variant {
	variants := r.catalog.ListVariants(asset.Path)
	for _, encoding := range encodings {
		for _, variant := range variants {
			if variant.Encoding != encoding {
				continue
			}
			if asset.SourceHash != "" && variant.SourceHash != "" && variant.SourceHash != asset.SourceHash {
				continue
			}
			if variant.ArtifactPath == "" {
				continue
			}
			if _, err := os.Stat(variant.ArtifactPath); err != nil {
				continue
			}
			return variant
		}
	}
	return nil
}

func (r *Resolver) pickImageVariant(asset *catalog.Asset, width int, formats []string) *catalog.Variant {
	sourceFormat := imageFormat(asset.MediaType)

	var candidates []*catalog.Variant
	for _, variant := range r.catalog.ListVariants(asset.Path) {
		if variant.Width <= 0 && variant.Format == "" {
			continue
		}
		if asset.SourceHash != "" && variant.SourceHash != "" && variant.SourceHash != asset.SourceHash {
			continue
		}
		if variant.ArtifactPath == "" {
			continue
		}
		if _, err := os.Stat(variant.ArtifactPath); err != nil {
			continue
		}
		candidates = append(candidates, variant)
	}
	if len(candidates) == 0 {
		return nil
	}

	if len(formats) == 0 {
		formats = []string{imageFormat(asset.MediaType)}
	}

	for _, format := range formats {
		var byFormat []*catalog.Variant
		for _, candidate := range candidates {
			candidateFormat := candidate.Format
			if candidateFormat == "" {
				candidateFormat = sourceFormat
			}
			if format == "" || candidateFormat == format {
				byFormat = append(byFormat, candidate)
			}
		}
		if len(byFormat) == 0 {
			continue
		}

		if width <= 0 {
			for _, candidate := range byFormat {
				if candidate.Width == 0 {
					return candidate
				}
			}
			continue
		}

		sort.Slice(byFormat, func(i, j int) bool {
			return byFormat[i].Width < byFormat[j].Width
		})
		for _, candidate := range byFormat {
			if candidate.Width >= width {
				return candidate
			}
		}
		return byFormat[len(byFormat)-1]
	}

	if width <= 0 {
		return nil
	}
	return nil
}

func candidates(requestPath, entry string) []string {
	normalized := normalizeAssetPath(requestPath)
	if normalized == "" {
		return []string{entry}
	}

	result := []string{normalized}
	if strings.HasSuffix(strings.TrimSpace(requestPath), "/") || path.Ext(normalized) == "" {
		result = append(result, path.Join(normalized, entry))
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

func parseAcceptEncoding(header string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	explicit := collectionmapping.NewMapWithCapacity[string, float64](4)
	wildcardQ := 0.0
	hasWildcard := false
	for _, rawPart := range strings.Split(header, ",") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		pieces := strings.Split(part, ";")
		encoding := strings.ToLower(strings.TrimSpace(pieces[0]))
		if encoding == "" {
			continue
		}

		q := 1.0
		for _, rawParam := range pieces[1:] {
			param := strings.TrimSpace(rawParam)
			if !strings.HasPrefix(strings.ToLower(param), "q=") {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(param), "q="))
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			if parsed < 0 {
				parsed = 0
			}
			if parsed > 1 {
				parsed = 1
			}
			q = parsed
		}

		if encoding == "*" {
			hasWildcard = true
			wildcardQ = q
			continue
		}
		if oldQ, ok := explicit.Get(encoding); !ok || q > oldQ {
			explicit.Set(encoding, q)
		}
	}

	type candidate struct {
		encoding string
		q        float64
		priority int
	}

	supported := []string{"br", "gzip"}
	choices := make([]candidate, 0, len(supported))
	for index, encoding := range supported {
		q, ok := explicit.Get(encoding)
		if !ok {
			if !hasWildcard {
				continue
			}
			q = wildcardQ
		}
		if q <= 0 {
			continue
		}
		choices = append(choices, candidate{
			encoding: encoding,
			q:        q,
			priority: index,
		})
	}

	sort.SliceStable(choices, func(i, j int) bool {
		if choices[i].q == choices[j].q {
			return choices[i].priority < choices[j].priority
		}
		return choices[i].q > choices[j].q
	})

	out := make([]string, 0, len(choices))
	for _, choice := range choices {
		out = append(out, choice.encoding)
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := collectionset.NewSetWithCapacity[string](len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if seen.Contains(value) {
			continue
		}
		seen.Add(value)
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func preferredWidths(width int) []int {
	if width <= 0 {
		return nil
	}
	return []int{width}
}

func preferredImageFormats(acceptHeader string, explicitFormat string, sourceMediaType string) []string {
	if explicitFormat != "" {
		return []string{explicitFormat}
	}
	if !isImageMediaType(sourceMediaType) {
		return nil
	}
	return parseAcceptImageFormats(acceptHeader, imageFormat(sourceMediaType))
}

func normalizeImageFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	default:
		return ""
	}
}

func imageFormat(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	default:
		return ""
	}
}

func isImageMediaType(mediaType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/")
}

func parseAcceptImageFormats(header string, sourceFormat string) []string {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	explicit := collectionmapping.NewMapWithCapacity[string, float64](4)
	wildcardImageQ := 0.0
	hasWildcardImage := false
	wildcardAnyQ := 0.0
	hasWildcardAny := false

	for _, rawPart := range strings.Split(header, ",") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		pieces := strings.Split(part, ";")
		mediaType := strings.ToLower(strings.TrimSpace(pieces[0]))
		if mediaType == "" {
			continue
		}

		q := 1.0
		for _, rawParam := range pieces[1:] {
			param := strings.TrimSpace(rawParam)
			if !strings.HasPrefix(strings.ToLower(param), "q=") {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(param), "q="))
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				continue
			}
			if parsed < 0 {
				parsed = 0
			}
			if parsed > 1 {
				parsed = 1
			}
			q = parsed
		}

		switch mediaType {
		case "image/*":
			hasWildcardImage = true
			wildcardImageQ = q
		case "*/*":
			hasWildcardAny = true
			wildcardAnyQ = q
		case "image/jpeg", "image/jpg":
			if oldQ, ok := explicit.Get("jpeg"); !ok || q > oldQ {
				explicit.Set("jpeg", q)
			}
		case "image/png":
			if oldQ, ok := explicit.Get("png"); !ok || q > oldQ {
				explicit.Set("png", q)
			}
		}
	}

	type candidate struct {
		format   string
		q        float64
		priority int
	}

	supported := []string{"jpeg", "png"}
	candidates := make([]candidate, 0, len(supported))
	for index, format := range supported {
		q, ok := explicit.Get(format)
		if !ok {
			if hasWildcardImage {
				q = wildcardImageQ
			} else if hasWildcardAny {
				q = wildcardAnyQ
			} else {
				q = 0
			}
		}
		if q <= 0 {
			continue
		}
		priority := index
		if format == sourceFormat {
			priority = -1
		}
		candidates = append(candidates, candidate{
			format:   format,
			q:        q,
			priority: priority,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].q == candidates[j].q {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].q > candidates[j].q
	})

	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.format)
	}
	return out
}
