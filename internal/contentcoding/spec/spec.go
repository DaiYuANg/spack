// Package spec defines the supported content-coding names and normalization helpers.
package spec

import (
	"strings"

	"github.com/arcgolabs/collectionx"
	"github.com/samber/lo"
)

func DefaultNames() collectionx.List[string] {
	return collectionx.NewList[string]("br", "zstd", "gzip")
}

func IsSupported(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "br", "zstd", "gzip":
		return true
	default:
		return false
	}
}

func ParseNames(raw string) collectionx.List[string] {
	if strings.TrimSpace(raw) == "" {
		return collectionx.NewList[string]()
	}
	return NormalizeNames(collectionx.NewList[string](strings.Split(raw, ",")...))
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

	normalized := collectionx.FilterMapList[string, string](values, func(_ int, raw string) (string, bool) {
		name := strings.ToLower(strings.TrimSpace(raw))
		return name, IsSupported(name)
	})
	if normalized.IsEmpty() {
		return nil
	}
	return collectionx.NewList[string](lo.Uniq[string](normalized.Values())...)
}
