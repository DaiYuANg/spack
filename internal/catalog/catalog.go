// Package catalog stores asset and variant metadata in memory.
package catalog

import (
	"cmp"
	"errors"
	"maps"
	"strconv"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

var ErrAssetNotFound = errors.New("asset not found")

type Asset struct {
	Path       string            `json:"path"`
	FullPath   string            `json:"full_path"`
	Size       int64             `json:"size"`
	MediaType  string            `json:"media_type"`
	SourceHash string            `json:"source_hash"`
	ETag       string            `json:"etag"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func (a *Asset) GetMetadata() map[string]string {
	if a == nil {
		return nil
	}
	return a.Metadata
}

type Variant struct {
	ID           string            `json:"id"`
	AssetPath    string            `json:"asset_path"`
	ArtifactPath string            `json:"artifact_path"`
	Size         int64             `json:"size"`
	MediaType    string            `json:"media_type"`
	SourceHash   string            `json:"source_hash"`
	ETag         string            `json:"etag"`
	Encoding     string            `json:"encoding,omitempty"`
	Format       string            `json:"format,omitempty"`
	Width        int               `json:"width,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func (v *Variant) GetMetadata() map[string]string {
	if v == nil {
		return nil
	}
	return v.Metadata
}

type Entry struct {
	Asset    *Asset     `json:"asset"`
	Variants []*Variant `json:"variants,omitempty"`
}

type Snapshot struct {
	Assets []*Entry `json:"assets"`
}

type Catalog interface {
	UpsertAsset(asset *Asset) error
	UpsertVariant(variant *Variant) error
	DeleteVariantByArtifactPath(artifactPath string) bool
	FindAsset(assetPath string) (*Asset, bool)
	ListVariants(assetPath string) collectionx.List[*Variant]
	AllAssets() collectionx.List[*Asset]
	AssetCount() int
	VariantCount() int
	Snapshot() *Snapshot
}

type variantRef struct {
	assetPath string
	id        string
}

type InMemoryCatalog struct {
	mu            sync.RWMutex
	assets        collectionx.Map[string, *Asset]
	variants      collectionx.Table[string, string, *Variant]
	artifactIndex collectionx.Map[string, variantRef]
}

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

	byAsset := c.variants.Row(assetPath)
	if len(byAsset) == 0 {
		return collectionx.NewList[*Variant]()
	}

	out := collectionx.MapList(collectionx.NewList(lo.Values(byAsset)...), func(_ int, variant *Variant) *Variant {
		return cloneVariant(variant)
	})
	out.Sort(func(left, right *Variant) int {
		return cmp.Compare(left.ID, right.ID)
	})
	return out
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
				Variants: c.ListVariants(asset.Path).Values(),
			}
		}).Values(),
	}
}

func cloneAsset(asset *Asset) *Asset {
	if asset == nil {
		return nil
	}

	cloned := *asset
	cloned.Metadata = cloneMap(asset.Metadata)
	return &cloned
}

func cloneVariant(variant *Variant) *Variant {
	if variant == nil {
		return nil
	}

	cloned := *variant
	cloned.Metadata = cloneMap(variant.Metadata)
	return &cloned
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	return maps.Clone(src)
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
