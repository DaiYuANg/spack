package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
	"github.com/pquerna/cachecontrol/cacheobject"
)

const cacheControlRevalidate = "public, max-age=0, must-revalidate"

func applyResolvedHeaders(c fiber.Ctx, cfg *config.Config, result *resolver.Result, requestedFormat string) {
	lastModified, hasLastModified := resolvedLastModified(result)
	cacheControl := cacheControlForResult(cfg, result)

	c.Set(fiber.HeaderContentType, result.MediaType)
	c.Append(fiber.HeaderVary, fiber.HeaderAcceptEncoding)
	if shouldVaryAccept(result.Asset.MediaType, requestedFormat) {
		c.Append(fiber.HeaderVary, fiber.HeaderAccept)
	}
	if result.ETag != "" {
		c.Set(fiber.HeaderETag, result.ETag)
	}
	if result.ContentEncoding != "" {
		c.Set(fiber.HeaderContentEncoding, result.ContentEncoding)
	}
	if hasLastModified {
		c.Set(fiber.HeaderLastModified, lastModified.UTC().Format(http.TimeFormat))
	}
	c.Set(fiber.HeaderCacheControl, cacheControl)
	applyExpiresHeader(c, cacheControl, lastModified, hasLastModified)
}

func shouldSendNotModified(c fiber.Ctx, request resolver.Request) bool {
	if request.RangeRequested {
		return false
	}
	return c.Req().Fresh()
}

func cacheControlForResult(cfg *config.Config, result *resolver.Result) string {
	if result == nil || result.Variant == nil {
		return cacheControlRevalidate
	}

	maxAge := variantMaxAge(cfg.Compression, result.Variant)
	if maxAge <= 0 {
		return cacheControlRevalidate
	}
	return fmt.Sprintf("public, max-age=%d, immutable", int(maxAge.Seconds()))
}

func applyExpiresHeader(c fiber.Ctx, cacheControl string, lastModified time.Time, hasLastModified bool) {
	expiresAt, ok := resolveExpiresAt(cacheControl, lastModified, hasLastModified)
	if !ok {
		c.Response().Header.Del(fiber.HeaderExpires)
		return
	}
	c.Set(fiber.HeaderExpires, expiresAt.UTC().Format(http.TimeFormat))
}

func resolveExpiresAt(cacheControl string, lastModified time.Time, hasLastModified bool) (time.Time, bool) {
	headers := make(http.Header, 2)
	headers.Set(fiber.HeaderCacheControl, cacheControl)
	if hasLastModified {
		headers.Set(fiber.HeaderLastModified, lastModified.UTC().Format(http.TimeFormat))
	}

	_, expiresAt, err := cacheobject.UsingRequestResponse(nil, http.StatusOK, headers, false)
	if err != nil || expiresAt.IsZero() {
		return time.Time{}, false
	}
	return expiresAt, true
}

func variantMaxAge(cfg config.Compression, variant *catalog.Variant) time.Duration {
	if variant == nil {
		return 0
	}
	if variant.Encoding != "" {
		if maxAge, ok := cfg.NamespaceMaxAges().Get("encoding"); ok && maxAge > 0 {
			return maxAge
		}
		return cfg.ParsedMaxAge()
	}
	if variant.Width > 0 || strings.TrimSpace(variant.Format) != "" {
		if maxAge, ok := cfg.NamespaceMaxAges().Get("image"); ok && maxAge > 0 {
			return maxAge
		}
	}
	return 0
}

func resolvedLastModified(result *resolver.Result) (time.Time, bool) {
	if result == nil {
		return time.Time{}, false
	}
	if timestamp, ok := metadataModTime(result.Variant); ok {
		return timestamp, true
	}
	if timestamp, ok := metadataModTime(result.Asset); ok {
		return timestamp, true
	}
	if strings.TrimSpace(result.FilePath) == "" {
		return time.Time{}, false
	}
	info, err := os.Stat(result.FilePath)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

type metadataCarrier interface {
	GetMetadata() map[string]string
}

func metadataModTime(carrier metadataCarrier) (time.Time, bool) {
	if carrier == nil {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(carrier.GetMetadata()["mtime_unix"])
	if raw == "" {
		return time.Time{}, false
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0), true
}
