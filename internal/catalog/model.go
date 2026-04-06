package catalog

import (
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type Asset struct {
	Path       string                          `json:"path"`
	FullPath   string                          `json:"full_path"`
	Size       int64                           `json:"size"`
	MediaType  string                          `json:"media_type"`
	SourceHash string                          `json:"source_hash"`
	ETag       string                          `json:"etag"`
	Metadata   collectionx.Map[string, string] `json:"metadata,omitempty"`
}

func (a *Asset) GetMetadata() collectionx.Map[string, string] {
	if a == nil {
		return nil
	}
	return a.Metadata
}

type Variant struct {
	ID           string                          `json:"id"`
	AssetPath    string                          `json:"asset_path"`
	ArtifactPath string                          `json:"artifact_path"`
	Size         int64                           `json:"size"`
	MediaType    string                          `json:"media_type"`
	SourceHash   string                          `json:"source_hash"`
	ETag         string                          `json:"etag"`
	Encoding     string                          `json:"encoding,omitempty"`
	Format       string                          `json:"format,omitempty"`
	Width        int                             `json:"width,omitempty"`
	Metadata     collectionx.Map[string, string] `json:"metadata,omitempty"`
}

func (v *Variant) GetMetadata() collectionx.Map[string, string] {
	if v == nil {
		return nil
	}
	return v.Metadata
}

type Entry struct {
	Asset    *Asset                     `json:"asset"`
	Variants collectionx.List[*Variant] `json:"variants,omitempty"`
}

type Snapshot struct {
	Assets collectionx.List[*Entry] `json:"assets"`
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
