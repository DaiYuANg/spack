package resolver

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	contentcodingspec "github.com/daiyuang/spack/internal/contentcoding/spec"
	"github.com/daiyuang/spack/internal/media"
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
	entries := collectionx.NewList[acceptEntry]()
	remaining := header
	for {
		part, rest, found := strings.Cut(remaining, ",")
		if entry, ok := parseAcceptEntry(part); ok {
			entries.Add(entry)
		}
		if !found {
			return entries
		}
		remaining = rest
	}
}

func parseAcceptEntry(rawPart string) (acceptEntry, bool) {
	part := strings.TrimSpace(rawPart)
	if part == "" {
		return acceptEntry{}, false
	}

	tokenRaw, paramsRaw, _ := strings.Cut(part, ";")
	token := strings.ToLower(strings.TrimSpace(tokenRaw))
	if token == "" {
		return acceptEntry{}, false
	}
	return acceptEntry{
		token: token,
		q:     parseAcceptQuality(paramsRaw),
	}, true
}

func parseAcceptQuality(params string) float64 {
	if strings.TrimSpace(params) == "" {
		return 1.0
	}

	quality := 1.0
	remaining := params
	for {
		paramRaw, rest, found := strings.Cut(remaining, ";")
		param := strings.TrimSpace(paramRaw)
		if len(param) >= 2 && (param[0] == 'q' || param[0] == 'Q') && param[1] == '=' {
			quality = clampAcceptQuality(strings.TrimSpace(param[2:]))
		}
		if !found {
			return quality
		}
		remaining = rest
	}
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
	prefs := encodingPreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	}
	entries.Range(func(_ int, entry acceptEntry) bool {
		if entry.token == "*" {
			prefs.hasWildcard = true
			prefs.wildcardQ = entry.q
			return true
		}
		if entry.q > prefs.explicit.GetOrDefault(entry.token, -1) {
			prefs.explicit.Set(entry.token, entry.q)
		}
		return true
	})
	return prefs
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
	prefs := imagePreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	}
	entries.Range(func(_ int, entry acceptEntry) bool {
		applyImagePreference(&prefs, entry)
		return true
	})
	return prefs
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
