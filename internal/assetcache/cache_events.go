package assetcache

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/eventx"
	appEvent "github.com/daiyuang/spack/internal/event"
)

func (c *Cache) start(_ context.Context) error {
	if c == nil || c.bus == nil || !c.Enabled() || c.variantRemovedUnsubscribe != nil {
		return nil
	}

	unsubscribe, err := eventx.Subscribe(c.bus, func(_ context.Context, event appEvent.VariantRemoved) error {
		c.Delete(event.ArtifactPath)
		return nil
	})
	if err != nil {
		return fmt.Errorf("subscribe variant removed: %w", err)
	}
	c.variantRemovedUnsubscribe = unsubscribe
	return nil
}

func (c *Cache) stop(_ context.Context) error {
	if c == nil || c.variantRemovedUnsubscribe == nil {
		return nil
	}
	c.variantRemovedUnsubscribe()
	c.variantRemovedUnsubscribe = nil
	return nil
}
