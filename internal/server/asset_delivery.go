package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
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

	if size, ok := resolvedAssetSizeOption(result).Get(); ok {
		if bodyCache.ShouldServe(size, request.RangeRequested) {
			return sendCachedResolvedAsset(c, bodyCache, result)
		}
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
	if found {
		return deliveryMemoryCacheHit, nil
	}
	return deliveryMemoryCacheFill, nil
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
	if result.Variant != nil && result.Variant.Size >= 0 {
		return mo.Some(result.Variant.Size)
	}
	if result.Asset != nil && result.Asset.Size >= 0 {
		return mo.Some(result.Asset.Size)
	}
	return mo.None[int64]()
}

func logRequest(logger *slog.Logger, c fiber.Ctx, startedAt time.Time) {
	attrs := collectionx.NewList(
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", c.Response().StatusCode()),
		slog.Duration("duration", time.Since(startedAt)),
		slog.String("request_id", c.GetRespHeader("Request-ID")),
	)
	if delivery := getAssetDelivery(c); delivery != "" {
		attrs.Add(slog.String("delivery", delivery))
	}
	logger.LogAttrs(context.Background(), slog.LevelInfo, "HTTP request", attrs.Values()...)
}
