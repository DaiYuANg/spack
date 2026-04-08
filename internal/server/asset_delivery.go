package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

func sendResolvedAsset(
	c fiber.Ctx,
	cfg *config.Config,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	logger *slog.Logger,
	bodyCache *assetcache.Cache,
) (string, error) {
	applyResolvedHeaders(c, cfg, result, requestedFormat)
	if handled := handleConditionalAssetRequest(c, request); handled {
		return "", nil
	}

	if shouldServeFromMemoryCache(bodyCache, result, request) {
		return sendCachedResolvedAsset(c, bodyCache, result)
	}

	return sendResolvedAssetFile(c, cfg, request, result, requestedFormat, logger)
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

func sendCachedResolvedAsset(c fiber.Ctx, bodyCache *assetcache.Cache, result *resolver.Result) (string, error) {
	body, found, err := bodyCache.GetOrLoad(result.FilePath)
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
	cfg *config.Config,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	logger *slog.Logger,
) (string, error) {
	if err := c.SendFile(result.FilePath, fiber.SendFile{ByteRange: true}); err != nil {
		logger.Error("Send asset failed",
			slog.String("path", result.FilePath),
			slog.String("err", err.Error()),
		)
		return "", fiber.ErrInternalServerError
	}

	// Override Fiber's extension-derived headers so variant metadata stays authoritative.
	applyResolvedHeaders(c, cfg, result, requestedFormat)
	if request.RangeRequested {
		return deliverySendFileRange, nil
	}
	return deliverySendFile, nil
}

func resolvedAssetSizeOption(result *resolver.Result) mo.Option[int64] {
	if result == nil {
		return mo.None[int64]()
	}
	return firstPresentInt64(
		variantSizeOption(result.Variant),
		assetSizeOption(result.Asset),
	)
}

func logRequest(logger *slog.Logger, c fiber.Ctx, startedAt time.Time) {
	attrs := collectionx.NewList(
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", c.Response().StatusCode()),
		slog.Duration("duration", time.Since(startedAt)),
		slog.String("request_id", c.GetRespHeader(RequestIDHeader)),
	)
	if delivery := getAssetDelivery(c); delivery != "" {
		attrs.Add(slog.String("delivery", delivery))
	}
	logger.LogAttrs(c.RequestCtx(), slog.LevelInfo, "HTTP request", attrs.Values()...)
}

func shouldServeFromMemoryCache(bodyCache *assetcache.Cache, result *resolver.Result, request resolver.Request) bool {
	size, ok := resolvedAssetSizeOption(result).Get()
	return ok && bodyCache.ShouldServe(size, request.RangeRequested)
}

func nonNegativeSize(size int64) mo.Option[int64] {
	if size < 0 {
		return mo.None[int64]()
	}
	return mo.Some(size)
}

func variantSizeOption(variant *catalog.Variant) mo.Option[int64] {
	if variant == nil {
		return mo.None[int64]()
	}
	return nonNegativeSize(variant.Size)
}

func assetSizeOption(asset *catalog.Asset) mo.Option[int64] {
	if asset == nil {
		return mo.None[int64]()
	}
	return nonNegativeSize(asset.Size)
}

func firstPresentInt64(options ...mo.Option[int64]) mo.Option[int64] {
	return lo.Reduce(options, func(found mo.Option[int64], option mo.Option[int64], _ int) mo.Option[int64] {
		if found.IsPresent() {
			return found
		}
		return option
	}, mo.None[int64]())
}
