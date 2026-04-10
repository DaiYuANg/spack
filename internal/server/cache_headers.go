package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/mo"
)

type resolvedHeaderPlan struct {
	contentType     string
	contentLength   mo.Option[string]
	vary            string
	etag            mo.Option[string]
	contentEncoding mo.Option[string]
	lastModified    mo.Option[string]
	cacheControl    string
	expires         mo.Option[string]
}

func newResolvedHeaderPlan(
	policy cachepolicy.ResponsePolicy,
	result *resolver.Result,
	requestedFormat string,
) resolvedHeaderPlan {
	lastModified := resolvedLastModified(result)
	cacheControl := policy.CacheControl(result)
	contentLength := mo.None[string]()
	if size, ok := resolvedAssetSize(result); ok {
		contentLength = mo.Some(strconv.FormatInt(size, 10))
	}
	expires := mo.None[string]()

	if policy != nil {
		if expiresAt, ok := policy.ExpiresAt(cacheControl, lastModified.modTime.OrEmpty(), lastModified.modTime.IsPresent()); ok {
			expires = mo.Some(expiresAt.UTC().Format(http.TimeFormat))
		}
	}

	sourceMediaType := ""
	if result != nil && result.Asset != nil {
		sourceMediaType = result.Asset.MediaType
	}

	return resolvedHeaderPlan{
		contentType:     result.MediaType,
		contentLength:   contentLength,
		vary:            resolvedVaryHeader(sourceMediaType, requestedFormat),
		etag:            mo.EmptyableToOption(strings.TrimSpace(result.ETag)),
		contentEncoding: mo.EmptyableToOption(strings.TrimSpace(result.ContentEncoding)),
		lastModified:    lastModified.header,
		cacheControl:    cacheControl,
		expires:         expires,
	}
}

func (p resolvedHeaderPlan) Apply(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, p.contentType)
	c.Set(fiber.HeaderVary, p.vary)
	c.Set(fiber.HeaderCacheControl, p.cacheControl)
	applyOptionalHeader(c, fiber.HeaderContentLength, p.contentLength)
	applyOptionalHeader(c, fiber.HeaderETag, p.etag)
	applyOptionalHeader(c, fiber.HeaderContentEncoding, p.contentEncoding)
	applyOptionalHeader(c, fiber.HeaderLastModified, p.lastModified)
	applyOptionalHeader(c, fiber.HeaderExpires, p.expires)
}

func (p resolvedHeaderPlan) ApplySendFileOverrides(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, p.contentType)
	applyOptionalHeader(c, fiber.HeaderContentLength, p.contentLength)
	applyOptionalHeader(c, fiber.HeaderContentEncoding, p.contentEncoding)
	applyOptionalHeader(c, fiber.HeaderLastModified, p.lastModified)
}

func applyOptionalHeader(c fiber.Ctx, key string, value mo.Option[string]) {
	if headerValue, ok := value.Get(); ok {
		c.Set(key, headerValue)
		return
	}
	c.Response().Header.Del(key)
}

func shouldSendNotModified(c fiber.Ctx, request resolver.Request) bool {
	if request.RangeRequested {
		return false
	}
	return c.Req().Fresh()
}

type resolvedLastModifiedValue struct {
	modTime mo.Option[time.Time]
	header  mo.Option[string]
}

func resolvedLastModified(result *resolver.Result) resolvedLastModifiedValue {
	if result == nil {
		return resolvedLastModifiedValue{
			modTime: mo.None[time.Time](),
			header:  mo.None[string](),
		}
	}

	if value, ok := metadataLastModifiedValue(result.Variant.GetMetadata()); ok {
		return value
	}
	if value, ok := metadataLastModifiedValue(result.Asset.GetMetadata()); ok {
		return value
	}
	if modTime, ok := catalog.FileModTime(result.FilePath).Get(); ok {
		return resolvedLastModifiedValue{
			modTime: mo.Some(modTime),
			header:  mo.Some(modTime.UTC().Format(http.TimeFormat)),
		}
	}
	return resolvedLastModifiedValue{
		modTime: mo.None[time.Time](),
		header:  mo.None[string](),
	}
}

func metadataLastModifiedValue(metadata collectionx.Map[string, string]) (resolvedLastModifiedValue, bool) {
	modTime, hasModTime := catalog.MetadataModTime(metadata).Get()
	header, hasHeader := catalog.MetadataLastModifiedHTTP(metadata).Get()
	if !hasModTime && !hasHeader {
		return resolvedLastModifiedValue{}, false
	}

	value := resolvedLastModifiedValue{
		modTime: mo.None[time.Time](),
		header:  mo.None[string](),
	}
	if hasModTime {
		value.modTime = mo.Some(modTime)
	}
	if hasHeader {
		value.header = mo.Some(header)
	}
	return value, true
}

func resolvedVaryHeader(sourceMediaType, requestedFormat string) string {
	if shouldVaryAccept(sourceMediaType, requestedFormat) {
		return fiber.HeaderAcceptEncoding + ", " + fiber.HeaderAccept
	}
	return fiber.HeaderAcceptEncoding
}
