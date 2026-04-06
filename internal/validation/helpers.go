package validation

import (
	"cmp"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func ParseFlexibleDuration(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err == nil {
		if d > 0 {
			return d
		}
		return 0
	}

	seconds, secErr := strconv.ParseInt(raw, 10, 64)
	if secErr != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func ParseWidths(raw string) collectionx.List[int] {
	if strings.TrimSpace(raw) == "" {
		return collectionx.NewList[int]()
	}

	widths := collectionx.FilterMapList(collectionx.NewList(strings.Split(raw, ",")...), func(_ int, part string) (int, bool) {
		width, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || width <= 0 {
			return 0, false
		}
		return width, true
	})
	if widths.IsEmpty() {
		return widths
	}

	widths.Sort(cmp.Compare[int])
	return collectionx.NewList(collectionx.NewOrderedSet(widths.Values()...).Values()...)
}
