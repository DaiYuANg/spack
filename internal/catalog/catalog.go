package catalog

import (
	"errors"
	"sort"
	"strconv"
	"sync"
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
	ListVariants(assetPath string) []*Variant
	AllAssets() []*Asset
	AssetCount() int
	VariantCount() int
	Snapshot() *Snapshot
}

type InMemoryCatalog struct {
	mu       sync.RWMutex
	assets   map[string]*Asset
	variants map[string]map[string]*Variant
}

func NewInMemoryCatalog() *InMemoryCatalog {
	return &InMemoryCatalog{
		assets:   make(map[string]*Asset),
		variants: make(map[string]map[string]*Variant),
	}
}

func (c *InMemoryCatalog) UpsertAsset(asset *Asset) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.assets[asset.Path] = cloneAsset(asset)
	return nil
}

func (c *InMemoryCatalog) UpsertVariant(variant *Variant) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.assets[variant.AssetPath]; !ok {
		return ErrAssetNotFound
	}

	id := variant.ID
	if id == "" {
		id = defaultVariantID(variant)
	}

	if _, ok := c.variants[variant.AssetPath]; !ok {
		c.variants[variant.AssetPath] = make(map[string]*Variant)
	}

	cloned := cloneVariant(variant)
	cloned.ID = id
	c.variants[variant.AssetPath][id] = cloned
	return nil
}

func (c *InMemoryCatalog) FindAsset(assetPath string) (*Asset, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	asset, ok := c.assets[assetPath]
	return cloneAsset(asset), ok
}

func (c *InMemoryCatalog) DeleteVariantByArtifactPath(artifactPath string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for assetPath, byAsset := range c.variants {
		for id, variant := range byAsset {
			if variant.ArtifactPath != artifactPath {
				continue
			}
			delete(byAsset, id)
			if len(byAsset) == 0 {
				delete(c.variants, assetPath)
			}
			return true
		}
	}
	return false
}

func (c *InMemoryCatalog) ListVariants(assetPath string) []*Variant {
	c.mu.RLock()
	defer c.mu.RUnlock()

	byAsset, ok := c.variants[assetPath]
	if !ok {
		return nil
	}

	out := make([]*Variant, 0, len(byAsset))
	for _, variant := range byAsset {
		out = append(out, cloneVariant(variant))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (c *InMemoryCatalog) AllAssets() []*Asset {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*Asset, 0, len(c.assets))
	for _, asset := range c.assets {
		out = append(out, cloneAsset(asset))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func (c *InMemoryCatalog) AssetCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.assets)
}

func (c *InMemoryCatalog) VariantCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := 0
	for _, variants := range c.variants {
		total += len(variants)
	}
	return total
}

func (c *InMemoryCatalog) Snapshot() *Snapshot {
	assets := c.AllAssets()
	entries := make([]*Entry, 0, len(assets))
	for _, asset := range assets {
		entries = append(entries, &Entry{
			Asset:    asset,
			Variants: c.ListVariants(asset.Path),
		})
	}
	return &Snapshot{Assets: entries}
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

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
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
