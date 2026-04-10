package catalog

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/mo"
)

const MetadataModTimeUnixKey = "mtime_unix"

func CloneMetadata(metadata collectionx.Map[string, string]) collectionx.Map[string, string] {
	if metadata == nil {
		return collectionx.NewMap[string, string]()
	}
	return metadata.Clone()
}

func MetadataModTime(metadata collectionx.Map[string, string]) mo.Option[time.Time] {
	if metadata == nil {
		return mo.None[time.Time]()
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

func MetadataWithModTime(metadata collectionx.Map[string, string], modTime time.Time) collectionx.Map[string, string] {
	cloned := CloneMetadata(metadata)
	if modTime.IsZero() {
		return cloned
	}

	cloned.Set(MetadataModTimeUnixKey, strconv.FormatInt(modTime.Unix(), 10))
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

func EnsureMetadataModTime(metadata collectionx.Map[string, string], path string) collectionx.Map[string, string] {
	cloned := CloneMetadata(metadata)
	if MetadataModTime(cloned).IsPresent() {
		return cloned
	}

	if modTime, ok := FileModTime(path).Get(); ok {
		cloned.Set(MetadataModTimeUnixKey, strconv.FormatInt(modTime.Unix(), 10))
	}

	return cloned
}
