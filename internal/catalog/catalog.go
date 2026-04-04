// Package catalog stores asset and variant metadata in memory.
package catalog

import (
	"cmp"
	"errors"
	"maps"
	"strconv"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
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

type InMemoryCatalog struct {
	mu       sync.RWMutex
	assets   collectionx.Map[string, *Asset]
	variants collectionx.Map[string, collectionx.Map[string, *Variant]]
}

func NewInMemoryCatalog() *InMemoryCatalog {
	return &InMemoryCatalog{
		assets:   collectionx.NewMap[string, *Asset](),
		variants: collectionx.NewMap[string, collectionx.Map[string, *Variant]](),
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

	byAsset, ok := c.variants.Get(variant.AssetPath)
	if !ok {
		byAsset = collectionx.NewMap[string, *Variant]()
		c.variants.Set(variant.AssetPath, byAsset)
	}

	cloned := cloneVariant(variant)
	cloned.ID = id
	byAsset.Set(id, cloned)
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

	assetPath, byAsset, ok := c.variants.FirstEntryWhere(func(_ string, byAsset collectionx.Map[string, *Variant]) bool {
		return byAsset != nil && byAsset.AnyEntryMatch(func(_ string, variant *Variant) bool {
			return variant.ArtifactPath == artifactPath
		})
	})
	if !ok || byAsset == nil {
		return false
	}

	id, _, ok := byAsset.FirstEntryWhere(func(_ string, variant *Variant) bool {
		return variant.ArtifactPath == artifactPath
	})
	if !ok {
		return false
	}

	byAsset.Delete(id)
	if byAsset.Len() == 0 {
		c.variants.Delete(assetPath)
	}
	return true
}

func (c *InMemoryCatalog) ListVariants(assetPath string) collectionx.List[*Variant] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	byAsset, ok := c.variants.Get(assetPath)
	if !ok {
		return collectionx.NewList[*Variant]()
	}

	out := collectionx.MapList(collectionx.NewList(byAsset.Values()...), func(_ int, variant *Variant) *Variant {
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

	return collectionx.ReduceList(collectionx.NewList(c.variants.Values()...), 0, func(total int, _ int, variants collectionx.Map[string, *Variant]) int {
		if variants == nil {
			return total
		}
		return total + variants.Len()
	})
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
