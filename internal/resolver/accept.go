package resolver

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	contentcodingspec "github.com/daiyuang/spack/internal/contentcoding/spec"
	"github.com/daiyuang/spack/internal/media"
	"github.com/samber/lo"
)

type acceptEntry struct {
	token string
	q     float64
}

type encodingPreferences struct {
	explicit    collectionx.Map[string, float64]
	wildcardQ   float64
	hasWildcard bool
}

type imagePreferences struct {
	explicit         collectionx.Map[string, float64]
	wildcardImageQ   float64
	hasWildcardImage bool
	wildcardAnyQ     float64
	hasWildcardAny   bool
}

type imagePreferenceMatch int

const (
	imagePreferenceNone imagePreferenceMatch = iota
	imagePreferenceAnyWildcard
	imagePreferenceImageWildcard
	imagePreferenceExplicit
)

func parseAcceptEncoding(header string, supported collectionx.List[string]) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return collectionx.NewList[string]()
	}

	return buildEncodingCandidates(collectEncodingPreferences(parseAcceptEntries(header)), supported)
}

func parseAcceptImageFormats(header, sourceFormat string, supported collectionx.List[string]) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return collectionx.NewList[string]()
	}

	return buildImageCandidates(collectImagePreferences(parseAcceptEntries(header)), sourceFormat, supported)
}

func parseAcceptEntries(header string) collectionx.List[acceptEntry] {
	return collectionx.FilterMapList(collectionx.NewList(strings.Split(header, ",")...), func(_ int, rawPart string) (acceptEntry, bool) {
		return parseAcceptEntry(rawPart)
	})
}

func parseAcceptEntry(rawPart string) (acceptEntry, bool) {
	part := strings.TrimSpace(rawPart)
	if part == "" {
		return acceptEntry{}, false
	}

	pieces := strings.Split(part, ";")
	token := strings.ToLower(strings.TrimSpace(pieces[0]))
	if token == "" {
		return acceptEntry{}, false
	}
	return acceptEntry{
		token: token,
		q:     parseAcceptQuality(pieces[1:]),
	}, true
}

func parseAcceptQuality(params []string) float64 {
	return lo.Reduce(params, func(q float64, rawParam string, _ int) float64 {
		param := strings.TrimSpace(rawParam)
		if !strings.HasPrefix(strings.ToLower(param), "q=") {
			return q
		}
		return clampAcceptQuality(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(param), "q=")))
	}, 1.0)
}

func clampAcceptQuality(raw string) float64 {
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 1.0
	}
	if parsed < 0 {
		return 0
	}
	if parsed > 1 {
		return 1
	}
	return parsed
}

func collectEncodingPreferences(entries collectionx.List[acceptEntry]) encodingPreferences {
	return lo.Reduce(entries.Values(), func(prefs encodingPreferences, entry acceptEntry, _ int) encodingPreferences {
		if entry.token == "*" {
			prefs.hasWildcard = true
			prefs.wildcardQ = entry.q
			return prefs
		}
		if entry.q > prefs.explicit.GetOrDefault(entry.token, -1) {
			prefs.explicit.Set(entry.token, entry.q)
		}
		return prefs
	}, encodingPreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	})
}

func buildEncodingCandidates(prefs encodingPreferences, supported collectionx.List[string]) collectionx.List[string] {
	type candidate struct {
		encoding string
		q        float64
		priority int
	}

	supported = contentcodingspec.NormalizeNames(supported)
	if supported.IsEmpty() {
		supported = contentcodingspec.DefaultNames()
	}
	choices := collectionx.FilterMapList(supported, func(index int, encoding string) (candidate, bool) {
		q, ok := encodingQuality(prefs, encoding)
		if !ok {
			return candidate{}, false
		}
		return candidate{
			encoding: encoding,
			q:        q,
			priority: index,
		}, true
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

func encodingQuality(prefs encodingPreferences, encoding string) (float64, bool) {
	if q, ok := prefs.explicit.Get(encoding); ok {
		return q, q > 0
	}
	if !prefs.hasWildcard || prefs.wildcardQ <= 0 {
		return 0, false
	}
	return prefs.wildcardQ, true
}

func collectImagePreferences(entries collectionx.List[acceptEntry]) imagePreferences {
	return lo.Reduce(entries.Values(), func(prefs imagePreferences, entry acceptEntry, _ int) imagePreferences {
		applyImagePreference(&prefs, entry)
		return prefs
	}, imagePreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	})
}

func applyImagePreference(prefs *imagePreferences, entry acceptEntry) {
	switch entry.token {
	case "image/*":
		prefs.hasWildcardImage = true
		prefs.wildcardImageQ = entry.q
	case "*/*":
		prefs.hasWildcardAny = true
		prefs.wildcardAnyQ = entry.q
	default:
		if descriptor, ok := media.LookupImageDescriptorByAcceptToken(entry.token); ok {
			setMaxQuality(prefs.explicit, descriptor.Name, entry.q)
		}
	}
}

func setMaxQuality(values collectionx.Map[string, float64], key string, q float64) {
	if q <= values.GetOrDefault(key, -1) {
		return
	}
	values.Set(key, q)
}

func buildImageCandidates(prefs imagePreferences, sourceFormat string, supported collectionx.List[string]) collectionx.List[string] {
	type candidate struct {
		format   string
		q        float64
		match    imagePreferenceMatch
		priority int
	}

	supported = imageFormatCandidates(supported, sourceFormat)
	candidates := collectionx.FilterMapList(supported, func(index int, format string) (candidate, bool) {
		q, match := imageQualityForFormat(prefs, format)
		if q <= 0 || match == imagePreferenceNone {
			return candidate{}, false
		}
		return candidate{
			format:   format,
			q:        q,
			match:    match,
			priority: imagePriority(index, format, sourceFormat),
		}, true
	})

	candidates.Sort(func(left, right candidate) int {
		if left.match != right.match {
			return cmp.Compare(int(right.match), int(left.match))
		}
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

func imageFormatCandidates(supported collectionx.List[string], sourceFormat string) collectionx.List[string] {
	candidates := collectionx.NewList[string]()
	if supported != nil && !supported.IsEmpty() {
		candidates.Add(supported.Values()...)
	}
	if sourceFormat != "" {
		candidates.Add(sourceFormat)
	}
	if candidates.IsEmpty() {
		candidates.Add(media.SupportedImageFormats().Values()...)
	}
	return media.NormalizeImageFormats(candidates)
}

func imageQualityForFormat(prefs imagePreferences, format string) (float64, imagePreferenceMatch) {
	if q, ok := prefs.explicit.Get(format); ok {
		return q, imagePreferenceExplicit
	}
	if prefs.hasWildcardImage {
		return prefs.wildcardImageQ, imagePreferenceImageWildcard
	}
	if prefs.hasWildcardAny {
		return prefs.wildcardAnyQ, imagePreferenceAnyWildcard
	}
	return 0, imagePreferenceNone
}

func imagePriority(index int, format, sourceFormat string) int {
	if format == sourceFormat {
		return -1
	}
	return index
}
