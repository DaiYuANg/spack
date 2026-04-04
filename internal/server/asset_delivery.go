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
	if shouldSendNotModified(c, request) {
		c.Status(fiber.StatusNotModified)
		return "", nil
	}

	if bodyCache.ShouldServe(resolvedAssetSize(result), request.RangeRequested) {
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

func resolvedAssetSize(result *resolver.Result) int64 {
	if result == nil {
		return -1
	}
	if result.Variant != nil && result.Variant.Size >= 0 {
		return result.Variant.Size
	}
	if result.Asset != nil {
		return result.Asset.Size
	}
	return -1
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
