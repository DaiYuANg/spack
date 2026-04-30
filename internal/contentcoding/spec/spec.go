// Package spec defines the supported content-coding names and normalization helpers.
package spec

import (
	"strings"

	cxlist "github.com/arcgolabs/collectionx/list"
	"github.com/samber/lo"
)

func DefaultNames() *cxlist.List[string] {
	return cxlist.NewList[string]("br", "zstd", "gzip")
}

func IsSupported(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "br", "zstd", "gzip":
		return true
	default:
		return false
	}
}

func ParseNames(raw string) *cxlist.List[string] {
	if strings.TrimSpace(raw) == "" {
		return cxlist.NewList[string]()
	}
	return NormalizeNames(cxlist.NewList[string](strings.Split(raw, ",")...))
}

func ResolveNames(raw string) *cxlist.List[string] {
	names := ParseNames(raw)
	if names.IsEmpty() {
		return DefaultNames()
	}
	return names
}

func NormalizeNames(values *cxlist.List[string]) *cxlist.List[string] {
	if values == nil || values.IsEmpty() {
		return nil
	}

	normalized := cxlist.MapList[string, string](values, func(_ int, raw string) string {
		return strings.ToLower(strings.TrimSpace(raw))
	}).Where(func(_ int, name string) bool {
		return IsSupported(name)
	})
	if normalized.IsEmpty() {
		return nil
	}
	return cxlist.NewList[string](lo.Uniq[string](normalized.Values())...)
}
