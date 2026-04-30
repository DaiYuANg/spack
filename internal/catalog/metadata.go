package catalog

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	cxmapping "github.com/arcgolabs/collectionx/mapping"
	"github.com/samber/mo"
)

const MetadataModTimeUnixKey = "mtime_unix"
const MetadataModTimeUnixNanoKey = "mtime_unix_nano"
const MetadataLastModifiedHTTPKey = "last_modified_http"

func CloneMetadata(metadata *cxmapping.Map[string, string]) *cxmapping.Map[string, string] {
	if metadata == nil {
		return cxmapping.NewMap[string, string]()
	}
	return metadata.Clone()
}

func MetadataModTime(metadata *cxmapping.Map[string, string]) mo.Option[time.Time] {
	if metadata == nil {
		return mo.None[time.Time]()
	}

	rawNanos, hasNanos := metadata.Get(MetadataModTimeUnixNanoKey)
	if hasNanos {
		nanos, err := strconv.ParseInt(strings.TrimSpace(rawNanos), 10, 64)
		if err == nil && nanos > 0 {
			return mo.Some(time.Unix(0, nanos))
		}
	}

	raw, ok := metadata.Get(MetadataModTimeUnixKey)
	if !ok {
		return mo.None[time.Time]()
	}

	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || seconds <= 0 {
		return mo.None[time.Time]()
	}

	return mo.Some(time.Unix(seconds, 0))
}

func MetadataLastModifiedHTTP(metadata *cxmapping.Map[string, string]) mo.Option[string] {
	if metadata == nil {
		return mo.None[string]()
	}

	if raw, ok := metadata.Get(MetadataLastModifiedHTTPKey); ok {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			return mo.Some(trimmed)
		}
	}

	if modTime, ok := MetadataModTime(metadata).Get(); ok {
		return mo.Some(modTime.UTC().Format(http.TimeFormat))
	}
	return mo.None[string]()
}

func MetadataWithModTime(metadata *cxmapping.Map[string, string], modTime time.Time) *cxmapping.Map[string, string] {
	cloned := CloneMetadata(metadata)
	if modTime.IsZero() {
		return cloned
	}

	setMetadataModTime(cloned, modTime)
	return cloned
}

func FileModTime(path string) mo.Option[time.Time] {
	if strings.TrimSpace(path) == "" {
		return mo.None[time.Time]()
	}

	info, err := os.Stat(path)
	if err != nil {
		return mo.None[time.Time]()
	}

	return mo.Some(info.ModTime())
}

func EnsureMetadataModTime(metadata *cxmapping.Map[string, string], path string) *cxmapping.Map[string, string] {
	cloned := CloneMetadata(metadata)
	if modTime, ok := MetadataModTime(cloned).Get(); ok {
		if !MetadataLastModifiedHTTP(cloned).IsPresent() {
			cloned.Set(MetadataLastModifiedHTTPKey, modTime.UTC().Format(http.TimeFormat))
		}
		return cloned
	}

	if modTime, ok := FileModTime(path).Get(); ok {
		setMetadataModTime(cloned, modTime)
	}

	return cloned
}

func setMetadataModTime(metadata *cxmapping.Map[string, string], modTime time.Time) {
	if metadata == nil || modTime.IsZero() {
		return
	}

	utc := modTime.UTC()
	metadata.Set(MetadataModTimeUnixKey, strconv.FormatInt(utc.Unix(), 10))
	metadata.Set(MetadataModTimeUnixNanoKey, strconv.FormatInt(utc.UnixNano(), 10))
	metadata.Set(MetadataLastModifiedHTTPKey, utc.Format(http.TimeFormat))
}
