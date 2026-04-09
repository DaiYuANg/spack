package server

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

func applyResolvedHeaders(
	c fiber.Ctx,
	policy cachepolicy.ResponsePolicy,
	result *resolver.Result,
	requestedFormat string,
) {
	lastModified, hasLastModified := resolvedLastModified(result)
	cacheControl := policy.CacheControl(result)

	c.Set(fiber.HeaderContentType, result.MediaType)
	if size, ok := resolvedAssetSize(result); ok {
		c.Set(fiber.HeaderContentLength, strconv.FormatInt(size, 10))
	}
	setResolvedVaryHeader(c, result.Asset.MediaType, requestedFormat)
	setIfNotEmpty(c, fiber.HeaderETag, result.ETag)
	setIfNotEmpty(c, fiber.HeaderContentEncoding, result.ContentEncoding)
	if hasLastModified {
		c.Set(fiber.HeaderLastModified, lastModified.UTC().Format(http.TimeFormat))
	}
	c.Set(fiber.HeaderCacheControl, cacheControl)
	applyExpiresHeader(c, policy, cacheControl, lastModified, hasLastModified)
}

func shouldSendNotModified(c fiber.Ctx, request resolver.Request) bool {
	if request.RangeRequested {
		return false
	}
	return c.Req().Fresh()
}

func applyExpiresHeader(
	c fiber.Ctx,
	policy cachepolicy.ResponsePolicy,
	cacheControl string,
	lastModified time.Time,
	hasLastModified bool,
) {
	if policy == nil {
		c.Response().Header.Del(fiber.HeaderExpires)
		return
	}

	expiresAt, ok := policy.ExpiresAt(cacheControl, lastModified, hasLastModified)
	if !ok {
		c.Response().Header.Del(fiber.HeaderExpires)
		return
	}
	c.Set(fiber.HeaderExpires, expiresAt.UTC().Format(http.TimeFormat))
}

func resolvedLastModified(result *resolver.Result) (time.Time, bool) {
	if result == nil {
		return time.Time{}, false
	}

	if modifiedAt, ok := metadataModTime(result.Variant); ok {
		return modifiedAt, true
	}
	if modifiedAt, ok := metadataModTime(result.Asset); ok {
		return modifiedAt, true
	}
	return fileModTime(result.FilePath)
}

type metadataCarrier interface {
	GetMetadata() collectionx.Map[string, string]
}

func metadataModTime(carrier metadataCarrier) (time.Time, bool) {
	if carrier == nil {
		return time.Time{}, false
	}
	value, ok := carrier.GetMetadata().Get("mtime_unix")
	if !ok {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, false
	}

	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0), true
}

func fileModTime(path string) (time.Time, bool) {
	if strings.TrimSpace(path) == "" {
		return time.Time{}, false
	}

	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

func setResolvedVaryHeader(c fiber.Ctx, sourceMediaType, requestedFormat string) {
	if shouldVaryAccept(sourceMediaType, requestedFormat) {
		c.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding+", "+fiber.HeaderAccept)
		return
	}
	c.Set(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
}

func setIfNotEmpty(c fiber.Ctx, key, value string) {
	if strings.TrimSpace(value) != "" {
		c.Set(key, value)
	}
}
