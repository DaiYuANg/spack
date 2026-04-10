package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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
	lastModifiedHeader := mo.None[string]()
	if value, ok := lastModified.Get(); ok {
		lastModifiedHeader = mo.Some(value.UTC().Format(http.TimeFormat))
	}
	expires := mo.None[string]()

	if policy != nil {
		if expiresAt, ok := policy.ExpiresAt(cacheControl, lastModified.OrEmpty(), lastModified.IsPresent()); ok {
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
		lastModified:    lastModifiedHeader,
		cacheControl:    cacheControl,
		expires:         expires,
	}
}

func (p resolvedHeaderPlan) Apply(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, p.contentType)
	c.Set(fiber.HeaderVary, p.vary)
	c.Set(fiber.HeaderCacheControl, p.cacheControl)
	p.contentLength.ForEach(func(value string) { c.Set(fiber.HeaderContentLength, value) })
	p.etag.ForEach(func(value string) { c.Set(fiber.HeaderETag, value) })
	p.contentEncoding.ForEach(func(value string) { c.Set(fiber.HeaderContentEncoding, value) })
	p.lastModified.ForEach(func(value string) { c.Set(fiber.HeaderLastModified, value) })

	if expiresAt, ok := p.expires.Get(); ok {
		c.Set(fiber.HeaderExpires, expiresAt)
		return
	}

	c.Response().Header.Del(fiber.HeaderExpires)
}

func (p resolvedHeaderPlan) ApplySendFileOverrides(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, p.contentType)
	p.contentLength.ForEach(func(value string) { c.Set(fiber.HeaderContentLength, value) })
	p.contentEncoding.ForEach(func(value string) { c.Set(fiber.HeaderContentEncoding, value) })
	p.lastModified.ForEach(func(value string) { c.Set(fiber.HeaderLastModified, value) })
}

func shouldSendNotModified(c fiber.Ctx, request resolver.Request) bool {
	if request.RangeRequested {
		return false
	}
	return c.Req().Fresh()
}

func resolvedLastModified(result *resolver.Result) mo.Option[time.Time] {
	if result == nil {
		return mo.None[time.Time]()
	}

	if modifiedAt := catalog.MetadataModTime(result.Variant.GetMetadata()); modifiedAt.IsPresent() {
		return modifiedAt
	}
	if modifiedAt := catalog.MetadataModTime(result.Asset.GetMetadata()); modifiedAt.IsPresent() {
		return modifiedAt
	}
	return catalog.FileModTime(result.FilePath)
}

func resolvedVaryHeader(sourceMediaType, requestedFormat string) string {
	if shouldVaryAccept(sourceMediaType, requestedFormat) {
		return fiber.HeaderAcceptEncoding + ", " + fiber.HeaderAccept
	}
	return fiber.HeaderAcceptEncoding
}
