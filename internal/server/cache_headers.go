package server

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

func applyResolvedHeaders(c fiber.Ctx, cfg *config.Config, result *resolver.Result, requestedFormat string) {
	policy := cachepolicy.NewResponsePolicy(&cfg.Compression)
	lastModified, hasLastModified := resolvedLastModified(result)
	cacheControl := policy.CacheControl(result)

	c.Set(fiber.HeaderContentType, result.MediaType)
	setIfPresent(c, fiber.HeaderContentLength, resolvedAssetSizeOption(result), func(size int64) string {
		return strconv.FormatInt(size, 10)
	})
	c.Append(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
	if shouldVaryAccept(result.Asset.MediaType, requestedFormat) {
		c.Append(fiber.HeaderVary, fiber.HeaderAccept)
	}
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
	return resolvedLastModifiedOption(result).Get()
}

type metadataCarrier interface {
	GetMetadata() collectionx.Map[string, string]
}

func resolvedLastModifiedOption(result *resolver.Result) mo.Option[time.Time] {
	if result == nil {
		return mo.None[time.Time]()
	}

	return firstPresentTime(
		metadataModTime(result.Variant),
		metadataModTime(result.Asset),
		fileModTime(result.FilePath),
	)
}

func metadataModTime(carrier metadataCarrier) mo.Option[time.Time] {
	if carrier == nil {
		return mo.None[time.Time]()
	}
	value, ok := carrier.GetMetadata().Get("mtime_unix")
	if !ok {
		return mo.None[time.Time]()
	}
	raw := strings.TrimSpace(value)
	if raw == "" {
		return mo.None[time.Time]()
	}

	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 {
		return mo.None[time.Time]()
	}
	return mo.Some(time.Unix(seconds, 0))
}

func fileModTime(path string) mo.Option[time.Time] {
	if strings.TrimSpace(path) == "" {
		return mo.None[time.Time]()
	}

	info, err := os.Stat(path)
	if err != nil {
		return mo.None[time.Time]()
	}
	return mo.Some(info.ModTime())
}

func firstPresentTime(options ...mo.Option[time.Time]) mo.Option[time.Time] {
	return lo.Reduce(options, func(found mo.Option[time.Time], option mo.Option[time.Time], _ int) mo.Option[time.Time] {
		if found.IsPresent() {
			return found
		}
		return option
	}, mo.None[time.Time]())
}

func setIfPresent[T any](c fiber.Ctx, key string, value mo.Option[T], format func(T) string) {
	if resolved, ok := value.Get(); ok {
		c.Set(key, format(resolved))
	}
}

func setIfNotEmpty(c fiber.Ctx, key, value string) {
	if strings.TrimSpace(value) != "" {
		c.Set(key, value)
	}
}
