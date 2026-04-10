// Package spec defines the supported content-coding names and normalization helpers.
package spec

import (
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

func DefaultNames() collectionx.List[string] {
	return collectionx.NewList("br", "zstd", "gzip")
}

func IsSupported(name string) bool {
	return lo.Contains(DefaultNames().Values(), strings.ToLower(strings.TrimSpace(name)))
}

func ParseNames(raw string) collectionx.List[string] {
	if strings.TrimSpace(raw) == "" {
		return collectionx.NewList[string]()
	}
	return NormalizeNames(collectionx.NewList(strings.Split(raw, ",")...))
}

func ResolveNames(raw string) collectionx.List[string] {
	names := ParseNames(raw)
	if names.IsEmpty() {
		return DefaultNames()
	}
	return names
}

func NormalizeNames(values collectionx.List[string]) collectionx.List[string] {
	if values == nil || values.IsEmpty() {
		return nil
	}

	normalized := lo.FilterMap(values.Values(), func(raw string, _ int) (string, bool) {
		name := strings.ToLower(strings.TrimSpace(raw))
		return name, IsSupported(name)
	})
	if len(normalized) == 0 {
		return nil
	}
	return collectionx.NewList(collectionx.NewOrderedSet(normalized...).Values()...)
}
