// Package requestpath normalizes incoming HTTP request paths for asset resolution.
package requestpath

import (
	"net/url"
	"path"
	"strings"
)

type Cleaned struct {
	Value               string
	AllowsEntryFallback bool
}

func Clean(raw string) Cleaned {
	trimmed := strings.TrimSpace(decode(raw))
	if trimmed == "" {
		return Cleaned{AllowsEntryFallback: true}
	}

	clean := path.Clean("/" + strings.TrimPrefix(trimmed, "/"))
	if clean == "/" || clean == "." {
		return Cleaned{AllowsEntryFallback: true}
	}

	value := strings.TrimPrefix(clean, "/")
	return Cleaned{
		Value:               value,
		AllowsEntryFallback: strings.HasSuffix(trimmed, "/") || path.Ext(value) == "",
	}
}

func CleanMounted(requestPath, mountPath string) Cleaned {
	return Clean(TrimMount(requestPath, mountPath))
}

func TrimMount(requestPath, mountPath string) string {
	mountPath = strings.TrimSpace(mountPath)
	if mountPath == "" || mountPath == "/" {
		return strings.TrimPrefix(requestPath, "/")
	}

	trimmed := strings.TrimPrefix(requestPath, strings.TrimRight(mountPath, "/"))
	return strings.TrimPrefix(trimmed, "/")
}

func decode(raw string) string {
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}
