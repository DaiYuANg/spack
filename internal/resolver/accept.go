package resolver

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
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

func parseAcceptEncoding(header string) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return collectionx.NewList[string]()
	}

	return buildEncodingCandidates(collectEncodingPreferences(parseAcceptEntries(header)))
}

func parseAcceptImageFormats(header, sourceFormat string) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return collectionx.NewList[string]()
	}

	return buildImageCandidates(collectImagePreferences(parseAcceptEntries(header)), sourceFormat)
}

func parseAcceptEntries(header string) []acceptEntry {
	entries := make([]acceptEntry, 0, 4)
	for rawPart := range strings.SplitSeq(header, ",") {
		entry, ok := parseAcceptEntry(rawPart)
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
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
	q := 1.0
	for _, rawParam := range params {
		param := strings.TrimSpace(rawParam)
		if !strings.HasPrefix(strings.ToLower(param), "q=") {
			continue
		}
		q = clampAcceptQuality(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(param), "q=")))
	}
	return q
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

func collectEncodingPreferences(entries []acceptEntry) encodingPreferences {
	prefs := encodingPreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	}
	for _, entry := range entries {
		if entry.token == "*" {
			prefs.hasWildcard = true
			prefs.wildcardQ = entry.q
			continue
		}
		if oldQ, ok := prefs.explicit.Get(entry.token); !ok || entry.q > oldQ {
			prefs.explicit.Set(entry.token, entry.q)
		}
	}
	return prefs
}

func buildEncodingCandidates(prefs encodingPreferences) collectionx.List[string] {
	type candidate struct {
		encoding string
		q        float64
		priority int
	}

	supported := collectionx.NewList("br", "gzip")
	choices := collectionx.NewListWithCapacity[candidate](supported.Len())
	supported.Range(func(index int, encoding string) bool {
		q, ok := encodingQuality(prefs, encoding)
		if !ok {
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

func encodingQuality(prefs encodingPreferences, encoding string) (float64, bool) {
	if q, ok := prefs.explicit.Get(encoding); ok {
		return q, q > 0
	}
	if !prefs.hasWildcard || prefs.wildcardQ <= 0 {
		return 0, false
	}
	return prefs.wildcardQ, true
}

func collectImagePreferences(entries []acceptEntry) imagePreferences {
	prefs := imagePreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	}
	for _, entry := range entries {
		applyImagePreference(&prefs, entry)
	}
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
	case "image/jpeg", "image/jpg":
		setMaxQuality(prefs.explicit, "jpeg", entry.q)
	case "image/png":
		setMaxQuality(prefs.explicit, "png", entry.q)
	}
}

func setMaxQuality(values collectionx.Map[string, float64], key string, q float64) {
	if oldQ, ok := values.Get(key); ok && q <= oldQ {
		return
	}
	values.Set(key, q)
}

func buildImageCandidates(prefs imagePreferences, sourceFormat string) collectionx.List[string] {
	type candidate struct {
		format   string
		q        float64
		priority int
	}

	supported := collectionx.NewList("jpeg", "png")
	candidates := collectionx.NewListWithCapacity[candidate](supported.Len())
	supported.Range(func(index int, format string) bool {
		q := imageQualityForFormat(prefs, format)
		if q <= 0 {
			return true
		}
		candidates.Add(candidate{
			format:   format,
			q:        q,
			priority: imagePriority(index, format, sourceFormat),
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

func imageQualityForFormat(prefs imagePreferences, format string) float64 {
	if q, ok := prefs.explicit.Get(format); ok {
		return q
	}
	if prefs.hasWildcardImage {
		return prefs.wildcardImageQ
	}
	if prefs.hasWildcardAny {
		return prefs.wildcardAnyQ
	}
	return 0
}

func imagePriority(index int, format, sourceFormat string) int {
	if format == sourceFormat {
		return -1
	}
	return index
}
