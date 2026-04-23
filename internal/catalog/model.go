package catalog

import "github.com/arcgolabs/collectionx"

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
	DeleteAsset(assetPath string) collectionx.List[*Variant]
	DeleteVariants(assetPath string) collectionx.List[*Variant]
	DeleteVariantByArtifactPath(artifactPath string) bool
	FindAsset(assetPath string) (*Asset, bool)
	FindEncodingVariant(assetPath, encoding string) (*Variant, bool)
	FindImageVariant(assetPath, format string, width int) (*Variant, bool)
	ListVariants(assetPath string) collectionx.List[*Variant]
	ListImageVariants(assetPath, format string) collectionx.List[*Variant]
	ListVariantsByStage(stage string) collectionx.List[*Variant]
	AllAssets() collectionx.List[*Asset]
	AllVariants() collectionx.List[*Variant]
	AssetCount() int
	VariantCount() int
	Snapshot() *Snapshot
}
