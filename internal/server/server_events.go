package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/eventx"
	appEvent "github.com/daiyuang/spack/internal/event"
	"github.com/daiyuang/spack/internal/resolver"
)

func publishVariantServed(
	ctx context.Context,
	result *resolver.Result,
	bus eventx.BusRuntime,
	logger *slog.Logger,
) {
	if result == nil || result.Variant == nil || bus == nil {
		return
	}
	assetPath := ""
	if result.Asset != nil {
		assetPath = result.Asset.Path
	}

	err := bus.Publish(ctx, appEvent.VariantServed{
		AssetPath:     assetPath,
		ArtifactPath:  result.FilePath,
		ServedAt:      time.Now(),
		ContentType:   result.MediaType,
		ContentCoding: result.ContentEncoding,
	})
	if err == nil || errors.Is(err, eventx.ErrBusClosed) {
		return
	}

	logger.Debug("Publish variant served event failed",
		slog.String("path", result.FilePath),
		slog.String("err", err.Error()),
	)
}
