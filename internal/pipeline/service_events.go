package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/daiyuang/spack/internal/catalog"
	appEvent "github.com/daiyuang/spack/internal/event"
)

func (s *Service) subscribeVariantServed() error {
	if s == nil || s.bus == nil || s.variantServedUnsubscribe != nil {
		return nil
	}

	unsubscribe, err := eventx.Subscribe(s.bus, func(_ context.Context, event appEvent.VariantServed) error {
		s.markVariantHitAt(event.ArtifactPath, event.ServedAt)
		return nil
	})
	if err != nil {
		return fmt.Errorf("subscribe variant served: %w", err)
	}
	s.variantServedUnsubscribe = unsubscribe
	return nil
}

func (s *Service) unsubscribeVariantServed() {
	if s == nil || s.variantServedUnsubscribe == nil {
		return
	}
	s.variantServedUnsubscribe()
	s.variantServedUnsubscribe = nil
}

func (s *Service) publishVariantRemoved(ctx context.Context, path string, reason appEvent.VariantRemovalReason) {
	if s == nil || s.bus == nil || strings.TrimSpace(path) == "" {
		return
	}

	err := s.bus.Publish(ctx, appEvent.VariantRemoved{
		ArtifactPath: path,
		Reason:       reason,
		RemovedAt:    time.Now(),
	})
	if err == nil || errors.Is(err, eventx.ErrBusClosed) {
		return
	}

	s.logger.Debug("Publish variant removed event failed",
		slog.String("path", path),
		slog.String("reason", string(reason)),
		slog.String("err", err.Error()),
	)
}

func (s *Service) publishVariantGenerated(ctx context.Context, stage string, variant *catalog.Variant) {
	if s == nil || s.bus == nil || variant == nil || strings.TrimSpace(variant.ArtifactPath) == "" {
		return
	}

	event := appEvent.VariantGenerated{
		AssetPath:    variant.AssetPath,
		ArtifactPath: variant.ArtifactPath,
		Stage:        strings.TrimSpace(stage),
		Size:         variant.Size,
		GeneratedAt:  time.Now(),
	}
	if err := s.bus.PublishAsync(ctx, event); err != nil {
		if errors.Is(err, eventx.ErrAsyncRuntimeUnavailable) {
			s.publishVariantGeneratedSync(ctx, event)
			return
		}
		if errors.Is(err, eventx.ErrBusClosed) {
			return
		}
		s.logger.Debug("Publish variant generated event failed",
			slog.String("path", variant.ArtifactPath),
			slog.String("stage", stage),
			slog.String("err", err.Error()),
		)
	}
}

func (s *Service) publishVariantGeneratedSync(ctx context.Context, event appEvent.VariantGenerated) {
	err := s.bus.Publish(ctx, event)
	if err == nil || errors.Is(err, eventx.ErrBusClosed) {
		return
	}

	s.logger.Debug("Publish variant generated event sync fallback failed",
		slog.String("path", event.ArtifactPath),
		slog.String("stage", event.Stage),
		slog.String("err", err.Error()),
	)
}
