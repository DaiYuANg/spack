package resolver

import (
	"cmp"
	"errors"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	PreferredEncodings collectionx.List[string]
	PreferredWidths    collectionx.List[int]
	PreferredFormats   collectionx.List[string]
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
	if request.Width > 0 || preferredImageFormats.Len() > 0 {
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

	if !request.RangeRequested && encodings.Len() > 0 {
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
	sourceFormat := imageFormat(asset.MediaType)
	candidates := r.catalog.ListVariants(asset.Path).Where(func(_ int, variant *catalog.Variant) bool {
		if variant.Width <= 0 && variant.Format == "" {
			return false
		}
		return isUsableVariant(variant, asset.SourceHash)
	})
	if candidates.IsEmpty() {
		return nil
	}

	if formats.IsEmpty() {
		formats = collectionx.NewList(sourceFormat)
	}

	var picked *catalog.Variant
	formats.Range(func(_ int, format string) bool {
		byFormat := candidates.Where(func(_ int, candidate *catalog.Variant) bool {
			return format == "" || variantFormat(candidate, sourceFormat) == format
		})
		if byFormat.IsEmpty() {
			return true
		}

		if width <= 0 {
			picked, _ = byFormat.FirstWhere(func(_ int, candidate *catalog.Variant) bool {
				return candidate.Width == 0
			}).Get()
			return picked == nil
		}

		byFormat.Sort(func(left, right *catalog.Variant) int {
			return cmp.Compare(left.Width, right.Width)
		})
		picked, _ = byFormat.FirstWhere(func(_ int, candidate *catalog.Variant) bool {
			return candidate.Width >= width
		}).Get()
		if picked != nil {
			return false
		}

		picked, _ = byFormat.GetLast()
		return picked == nil
	})
	return picked
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

func parseAcceptEncoding(header string) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return collectionx.NewList[string]()
	}

	explicit := collectionx.NewMapWithCapacity[string, float64](4)
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

	supported := collectionx.NewList("br", "gzip")
	choices := collectionx.NewListWithCapacity[candidate](supported.Len())
	supported.Range(func(index int, encoding string) bool {
		q, ok := explicit.Get(encoding)
		if !ok {
			if !hasWildcard {
				return true
			}
			q = wildcardQ
		}
		if q <= 0 {
			return true
		}
		choices.Add(candidate{
			encoding: encoding,
			q:        q,
			priority: index,
		})
		return true
	})

	choices.Sort(func(left, right candidate) int {
		if left.q == right.q {
			return cmp.Compare(left.priority, right.priority)
		}
		if left.q > right.q {
			return -1
		}
		return 1
	})

	return collectionx.MapList(choices, func(_ int, choice candidate) string {
		return choice.encoding
	})
}

func uniqueStrings(values collectionx.List[string]) collectionx.List[string] {
	ordered := collectionx.NewOrderedSetWithCapacity[string](values.Len())
	values.Each(func(_ int, value string) {
		if value == "" {
			return
		}
		ordered.Add(value)
	})
	return collectionx.NewList(ordered.Values()...)
}

func isUsableVariant(variant *catalog.Variant, assetSourceHash string) bool {
	if variant == nil || strings.TrimSpace(variant.ArtifactPath) == "" {
		return false
	}
	if assetSourceHash != "" && variant.SourceHash != "" && variant.SourceHash != assetSourceHash {
		return false
	}
	if _, err := os.Stat(variant.ArtifactPath); err != nil {
		return false
	}
	return true
}

func variantFormat(variant *catalog.Variant, sourceFormat string) string {
	if variant == nil || strings.TrimSpace(variant.Format) == "" {
		return sourceFormat
	}
	return variant.Format
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func preferredWidths(width int) collectionx.List[int] {
	if width <= 0 {
		return collectionx.NewList[int]()
	}
	return collectionx.NewList(width)
}

func preferredImageFormats(acceptHeader string, explicitFormat string, sourceMediaType string) collectionx.List[string] {
	if explicitFormat != "" {
		return collectionx.NewList(explicitFormat)
	}
	if !isImageMediaType(sourceMediaType) {
		return collectionx.NewList[string]()
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

func parseAcceptImageFormats(header string, sourceFormat string) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return collectionx.NewList[string]()
	}

	explicit := collectionx.NewMapWithCapacity[string, float64](4)
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

	supported := collectionx.NewList("jpeg", "png")
	candidates := collectionx.NewListWithCapacity[candidate](supported.Len())
	supported.Range(func(index int, format string) bool {
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
			return true
		}
		priority := index
		if format == sourceFormat {
			priority = -1
		}
		candidates.Add(candidate{
			format:   format,
			q:        q,
			priority: priority,
		})
		return true
	})

	candidates.Sort(func(left, right candidate) int {
		if left.q == right.q {
			return cmp.Compare(left.priority, right.priority)
		}
		if left.q > right.q {
			return -1
		}
		return 1
	})

	return collectionx.MapList(candidates, func(_ int, candidate candidate) string {
		return candidate.format
	})
}
