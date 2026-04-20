package resolver

import (
	"cmp"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	contentcodingspec "github.com/daiyuang/spack/internal/contentcoding/spec"
)

type encodingPreferences struct {
	explicit    collectionx.Map[string, float64]
	wildcardQ   float64
	hasWildcard bool
}

func parseAcceptEncoding(header string, supported collectionx.List[string]) collectionx.List[string] {
	if strings.TrimSpace(header) == "" {
		return nil
	}

	return buildEncodingCandidates(collectEncodingPreferences(header), supported)
}

func collectEncodingPreferences(header string) encodingPreferences {
	prefs := encodingPreferences{
		explicit: collectionx.NewMapWithCapacity[string, float64](4),
	}
	forEachAcceptEntry(header, func(entry acceptEntry) bool {
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
	choices := collectionx.FilterMapList[string, candidate](supported, func(index int, encoding string) (candidate, bool) {
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

	if choices.IsEmpty() {
		return nil
	}
	return collectionx.MapList[candidate, string](choices, func(_ int, choice candidate) string {
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
