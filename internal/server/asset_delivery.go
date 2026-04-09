package server

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
)

func sendResolvedAsset(
	c fiber.Ctx,
	responsePolicy cachepolicy.ResponsePolicy,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	logger *slog.Logger,
	bodyCache *assetcache.Cache,
) (string, error) {
	applyResolvedHeaders(c, responsePolicy, result, requestedFormat)
	if handled := handleConditionalAssetRequest(c, request); handled {
		return "", nil
	}

	if shouldServeFromMemoryCache(bodyCache, result, request) {
		return sendCachedResolvedAsset(c, bodyCache, result, request)
	}

	return sendResolvedAssetFile(c, responsePolicy, request, result, requestedFormat, logger)
}

func handleConditionalAssetRequest(c fiber.Ctx, request resolver.Request) bool {
	if shouldSendNotModified(c, request) {
		c.Status(fiber.StatusNotModified)
		return true
	}
	if c.Method() == fiber.MethodHead {
		c.Status(fiber.StatusOK)
		return true
	}
	return false
}

func sendCachedResolvedAsset(
	c fiber.Ctx,
	bodyCache *assetcache.Cache,
	result *resolver.Result,
	request resolver.Request,
) (string, error) {
	body, found, err := bodyCache.GetOrLoadWithRequest(result.FilePath, buildMemoryCacheRequest(result, request))
	if err != nil {
		return "", fiber.ErrInternalServerError
	}
	if err := c.Send(body); err != nil {
		return "", fmt.Errorf("send cached asset body: %w", err)
	}
	return lo.Ternary(found, deliveryMemoryCacheHit, deliveryMemoryCacheFill), nil
}

func sendResolvedAssetFile(
	c fiber.Ctx,
	responsePolicy cachepolicy.ResponsePolicy,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	logger *slog.Logger,
) (string, error) {
	if err := c.SendFile(result.FilePath, fiber.SendFile{ByteRange: true}); err != nil {
		if logger != nil {
			logger.Error("Send asset failed",
				slog.String("path", result.FilePath),
				slog.String("err", err.Error()),
			)
		}
		return "", fiber.ErrInternalServerError
	}

	// Override Fiber's extension-derived headers so variant metadata stays authoritative.
	applyResolvedHeaders(c, responsePolicy, result, requestedFormat)
	if request.RangeRequested {
		return deliverySendFileRange, nil
	}
	return deliverySendFile, nil
}

func resolvedAssetSize(result *resolver.Result) (int64, bool) {
	if result == nil {
		return 0, false
	}
	if size, ok := variantSize(result.Variant); ok {
		return size, true
	}
	return assetSize(result.Asset)
}

func logRequest(logger *slog.Logger, c fiber.Ctx, startedAt time.Time) {
	attrs := []slog.Attr{
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", c.Response().StatusCode()),
		slog.Duration("duration", time.Since(startedAt)),
		slog.String("request_id", c.GetRespHeader(RequestIDHeader)),
	}
	if delivery := getAssetDelivery(c); delivery != "" {
		attrs = append(attrs, slog.String("delivery", delivery))
	}
	logger.LogAttrs(c.RequestCtx(), slog.LevelInfo, "HTTP request", attrs...)
}

func shouldServeFromMemoryCache(bodyCache *assetcache.Cache, result *resolver.Result, request resolver.Request) bool {
	return bodyCache.ShouldServeRequest(buildMemoryCacheRequest(result, request))
}

func buildMemoryCacheRequest(result *resolver.Result, request resolver.Request) cachepolicy.MemoryRequest {
	if result == nil {
		return cachepolicy.MemoryRequest{
			RangeRequested: request.RangeRequested,
			UseCase:        cachepolicy.MemoryUseCaseServe,
		}
	}

	cacheRequest := cachepolicy.MemoryRequest{
		Path:           result.FilePath,
		MediaType:      result.MediaType,
		RangeRequested: request.RangeRequested,
		UseCase:        cachepolicy.MemoryUseCaseServe,
		Kind:           cachepolicy.MemoryEntryKindAsset,
	}

	if result.Asset != nil {
		cacheRequest.AssetPath = result.Asset.Path
		cacheRequest.Size = result.Asset.Size
		cacheRequest.MediaType = result.Asset.MediaType
	}

	if result.Variant != nil {
		cacheRequest.AssetPath = result.Variant.AssetPath
		cacheRequest.Size = result.Variant.Size
		cacheRequest.MediaType = firstNonEmptyString(result.Variant.MediaType, cacheRequest.MediaType)
		cacheRequest.Encoding = result.Variant.Encoding
		cacheRequest.Format = result.Variant.Format
		cacheRequest.Width = result.Variant.Width
		cacheRequest.Kind = cachepolicy.MemoryEntryKindVariant
	}

	return cacheRequest
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func nonNegativeSize(size int64) (int64, bool) {
	if size < 0 {
		return 0, false
	}
	return size, true
}

func variantSize(variant *catalog.Variant) (int64, bool) {
	if variant == nil {
		return 0, false
	}
	return nonNegativeSize(variant.Size)
}

func assetSize(asset *catalog.Asset) (int64, bool) {
	if asset == nil {
		return 0, false
	}
	return nonNegativeSize(asset.Size)
}
