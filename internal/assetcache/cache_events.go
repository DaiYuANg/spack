package assetcache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	appEvent "github.com/daiyuang/spack/internal/event"
)

func (c *Cache) start(_ context.Context) error {
	if c == nil || c.bus == nil || !c.Enabled() {
		return nil
	}
	if c.variantRemovedUnsubscribe != nil || c.variantGeneratedUnsubscribe != nil {
		return nil
	}

	unsubscribe, err := c.subscribeVariantRemoved()
	if err != nil {
		return err
	}
	c.variantRemovedUnsubscribe = unsubscribe

	generatedUnsubscribe, err := c.subscribeVariantGenerated()
	if err != nil {
		unsubscribe()
		c.variantRemovedUnsubscribe = nil
		return err
	}
	c.variantGeneratedUnsubscribe = generatedUnsubscribe
	return nil
}

func (c *Cache) stop(_ context.Context) error {
	if c == nil {
		return nil
	}
	if c.variantRemovedUnsubscribe != nil {
		c.variantRemovedUnsubscribe()
		c.variantRemovedUnsubscribe = nil
	}
	if c.variantGeneratedUnsubscribe != nil {
		c.variantGeneratedUnsubscribe()
		c.variantGeneratedUnsubscribe = nil
	}
	return nil
}

func (c *Cache) subscribeVariantRemoved() (func(), error) {
	unsubscribe, err := eventx.Subscribe(c.bus, func(_ context.Context, event appEvent.VariantRemoved) error {
		c.Delete(event.ArtifactPath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe variant removed: %w", err)
	}
	return unsubscribe, nil
}

func (c *Cache) subscribeVariantGenerated() (func(), error) {
	unsubscribe, err := eventx.Subscribe(c.bus, func(_ context.Context, event appEvent.VariantGenerated) error {
		if preloadErr := c.preloadPath(event.ArtifactPath, event.Size, nil); preloadErr != nil && c.logger != nil {
			c.logger.Debug("Preload generated variant failed",
				slog.String("path", event.ArtifactPath),
				slog.String("stage", event.Stage),
				slog.String("err", preloadErr.Error()),
			)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe variant generated: %w", err)
	}
	return unsubscribe, nil
}
