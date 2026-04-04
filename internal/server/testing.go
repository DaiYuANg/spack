package server

import (
	"github.com/DaiYuANg/arcgo/observabilityx"
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
