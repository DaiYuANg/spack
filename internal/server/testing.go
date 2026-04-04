package server

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

// ShouldVaryAcceptForTest exposes vary-header behavior for external tests.
func ShouldVaryAcceptForTest(sourceMediaType, explicitFormat string) bool {
	return shouldVaryAccept(sourceMediaType, explicitFormat)
}

// MetricsMiddlewareForTest exposes HTTP metrics middleware for external tests.
func MetricsMiddlewareForTest(obs observabilityx.Observability) fiber.Handler {
	return metricsMiddleware(obs)
}

// SetAssetDeliveryForTest exposes delivery tagging for external tests.
func SetAssetDeliveryForTest(c fiber.Ctx, delivery string) {
	setAssetDelivery(c, delivery)
}

// PublishVariantServedForTest exposes variant-served event publishing for external tests.
func PublishVariantServedForTest(
	ctx context.Context,
	result *resolver.Result,
	bus eventx.BusRuntime,
	logger *slog.Logger,
) {
	publishVariantServed(ctx, result, bus, logger)
}
