// Package catalog stores asset and variant metadata in memory.
package catalog

import (
	"cmp"
	"errors"
	"strconv"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

var ErrAssetNotFound = errors.New("asset not found")

func NewInMemoryCatalog() *InMemoryCatalog {
	return &InMemoryCatalog{
		assets:        collectionx.NewMap[string, *Asset](),
		variants:      collectionx.NewTable[string, string, *Variant](),
		artifactIndex: collectionx.NewMap[string, variantRef](),
	}
}

func (c *InMemoryCatalog) UpsertAsset(asset *Asset) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.assets.Set(asset.Path, cloneAsset(asset))
	return nil
}

func (c *InMemoryCatalog) UpsertVariant(variant *Variant) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.assets.Get(variant.AssetPath); !ok {
		return ErrAssetNotFound
	}

	id := variant.ID
	if id == "" {
		id = defaultVariantID(variant)
	}

	cloned := cloneVariant(variant)
	cloned.ID = id
	c.removeStaleVariantIndex(variant.AssetPath, id)
	c.removeConflictingArtifactIndex(cloned.ArtifactPath, variant.AssetPath, id)
	c.variants.Put(variant.AssetPath, id, cloned)
	c.indexArtifactPath(cloned.ArtifactPath, variant.AssetPath, id)
	return nil
}

func (c *InMemoryCatalog) FindAsset(assetPath string) (*Asset, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	asset, ok := c.assets.Get(assetPath)
	return cloneAsset(asset), ok
}

func (c *InMemoryCatalog) DeleteAsset(assetPath string) collectionx.List[*Variant] {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.assets.Delete(assetPath)
	return c.deleteVariantsLocked(assetPath)
}

func (c *InMemoryCatalog) DeleteVariants(assetPath string) collectionx.List[*Variant] {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.deleteVariantsLocked(assetPath)
}

func (c *InMemoryCatalog) DeleteVariantByArtifactPath(artifactPath string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ref, ok := c.artifactIndex.Get(artifactPath)
	if !ok {
		return false
	}

	c.artifactIndex.Delete(artifactPath)
	return c.variants.Delete(ref.assetPath, ref.id)
}

func (c *InMemoryCatalog) ListVariants(assetPath string) collectionx.List[*Variant] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return cloneVariantList(c.variants.Row(assetPath))
}

func (c *InMemoryCatalog) ListVariantsView(assetPath string) collectionx.List[*Variant] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return sortedVariantList(c.variants.Row(assetPath))
}

func (c *InMemoryCatalog) AllAssets() collectionx.List[*Asset] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := collectionx.MapList(collectionx.NewList(c.assets.Values()...), func(_ int, asset *Asset) *Asset {
		return cloneAsset(asset)
	})
	out.Sort(func(left, right *Asset) int {
		return cmp.Compare(left.Path, right.Path)
	})
	return out
}

func (c *InMemoryCatalog) AssetCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.assets.Len()
}

func (c *InMemoryCatalog) VariantCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.variants.Len()
}

func (c *InMemoryCatalog) Snapshot() *Snapshot {
	assets := c.AllAssets()
	return &Snapshot{
		Assets: collectionx.MapList(assets, func(_ int, asset *Asset) *Entry {
			return &Entry{
				Asset:    asset,
				Variants: c.ListVariants(asset.Path),
			}
		}),
	}
}

func cloneAsset(asset *Asset) *Asset {
	if asset == nil {
		return nil
	}

	cloned := *asset
	cloned.Metadata = asset.Metadata.Clone()
	return &cloned
}

func cloneVariant(variant *Variant) *Variant {
	if variant == nil {
		return nil
	}

	cloned := *variant
	cloned.Metadata = variant.Metadata
	return &cloned
}

func (c *InMemoryCatalog) removeStaleVariantIndex(assetPath, id string) {
	existing, ok := c.variants.Get(assetPath, id)
	if !ok || existing == nil {
		return
	}
	c.artifactIndex.Delete(existing.ArtifactPath)
}

func (c *InMemoryCatalog) removeConflictingArtifactIndex(artifactPath, assetPath, id string) {
	if artifactPath == "" {
		return
	}

	ref, ok := c.artifactIndex.Get(artifactPath)
	if !ok || (ref.assetPath == assetPath && ref.id == id) {
		return
	}

	c.variants.Delete(ref.assetPath, ref.id)
}

func (c *InMemoryCatalog) indexArtifactPath(artifactPath, assetPath, id string) {
	if artifactPath == "" {
		return
	}
	c.artifactIndex.Set(artifactPath, variantRef{assetPath: assetPath, id: id})
}

func (c *InMemoryCatalog) deleteVariantsLocked(assetPath string) collectionx.List[*Variant] {
	byAsset := c.variants.Row(assetPath)
	if len(byAsset) == 0 {
		c.variants.DeleteRow(assetPath)
		return collectionx.NewList[*Variant]()
	}

	out := cloneVariantList(byAsset)

	out.Range(func(_ int, variant *Variant) bool {
		c.artifactIndex.Delete(variant.ArtifactPath)
		return true
	})
	c.variants.DeleteRow(assetPath)
	return out
}

func cloneVariantList(byAsset map[string]*Variant) collectionx.List[*Variant] {
	out := sortedVariantList(byAsset)
	return collectionx.MapList(out, func(_ int, variant *Variant) *Variant {
		return cloneVariant(variant)
	})
}

func sortedVariantList(byAsset map[string]*Variant) collectionx.List[*Variant] {
	if len(byAsset) == 0 {
		return collectionx.NewList[*Variant]()
	}

	out := collectionx.NewList(lo.Values(byAsset)...)
	out.Sort(func(left, right *Variant) int {
		return cmp.Compare(left.ID, right.ID)
	})
	return out
}

func defaultVariantID(variant *Variant) string {
	id := variant.AssetPath
	if variant.Encoding != "" {
		id += "|encoding=" + variant.Encoding
	}
	if variant.Format != "" {
		id += "|format=" + variant.Format
	}
	if variant.Width > 0 {
		id += "|width=" + strconv.Itoa(variant.Width)
	}
	return id
}
