// Package catalog stores asset and variant metadata in memory.
package catalog

import (
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/hashicorp/go-memdb"
)

const (
	catalogAssetsTable                  = "assets"
	catalogVariantsTable                = "variants"
	catalogVariantArtifactPathIndex     = "artifact_path"
	catalogVariantAssetEncodingIndex    = "asset_path_encoding"
	catalogVariantAssetFormatWidthIndex = "asset_path_format_width"
)

var ErrAssetNotFound = errors.New("asset not found")

func (c *InMemoryCatalog) UpsertAsset(asset *Asset) error {
	record := newAssetRecord(asset)

	txn := c.db.Txn(true)
	defer txn.Abort()

	if err := txn.Insert(catalogAssetsTable, record); err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (c *InMemoryCatalog) UpsertVariant(variant *Variant) error {
	id := variant.ID
	if id == "" {
		id = defaultVariantID(variant)
	}

	record := newVariantRecord(variant, id)
	txn := c.db.Txn(true)
	defer txn.Abort()

	if !assetExists(txn, record.AssetPath) {
		return ErrAssetNotFound
	}

	pendingDeletes := make(map[string]*variantRecord, 4)
	collectVariantDelete(txn, pendingDeletes, "id", record.AssetPath, record.ID)
	if record.ArtifactPath != "" {
		collectVariantDelete(txn, pendingDeletes, catalogVariantArtifactPathIndex, record.ArtifactPath)
	}
	if record.Encoding != "" {
		collectVariantDelete(txn, pendingDeletes, catalogVariantAssetEncodingIndex, record.AssetPath, record.Encoding)
	}
	if record.ImageFormat != "" {
		collectVariantDelete(txn, pendingDeletes, catalogVariantAssetFormatWidthIndex, record.AssetPath, record.ImageFormat, record.Width)
	}

	for _, existing := range pendingDeletes {
		if err := txn.Delete(catalogVariantsTable, existing); err != nil && !errors.Is(err, memdb.ErrNotFound) {
			return err
		}
	}
	if err := txn.Insert(catalogVariantsTable, record); err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (c *InMemoryCatalog) FindAsset(assetPath string) (*Asset, bool) {
	asset, ok := c.FindAssetView(assetPath)
	return cloneAsset(asset), ok
}

func (c *InMemoryCatalog) FindAssetView(assetPath string) (*Asset, bool) {
	txn := c.db.Txn(false)
	defer txn.Abort()

	record, ok := findAssetRecord(txn, assetPath)
	if !ok {
		return nil, false
	}
	return record.Asset, true
}

func (c *InMemoryCatalog) FindEncodingVariant(assetPath, encoding string) (*Variant, bool) {
	variant, ok := c.FindEncodingVariantView(assetPath, encoding)
	return cloneVariant(variant), ok
}

func (c *InMemoryCatalog) FindEncodingVariantView(assetPath, encoding string) (*Variant, bool) {
	txn := c.db.Txn(false)
	defer txn.Abort()

	record, ok := findVariantRecord(txn, catalogVariantAssetEncodingIndex, assetPath, encoding)
	if !ok {
		return nil, false
	}
	return record.Variant, true
}

func (c *InMemoryCatalog) FindImageVariant(assetPath, format string, width int) (*Variant, bool) {
	variant, ok := c.FindImageVariantView(assetPath, format, width)
	return cloneVariant(variant), ok
}

func (c *InMemoryCatalog) FindImageVariantView(assetPath, format string, width int) (*Variant, bool) {
	txn := c.db.Txn(false)
	defer txn.Abort()

	record, ok := findVariantRecord(txn, catalogVariantAssetFormatWidthIndex, assetPath, format, width)
	if !ok {
		return nil, false
	}
	return record.Variant, true
}

func (c *InMemoryCatalog) DeleteAsset(assetPath string) collectionx.List[*Variant] {
	txn := c.db.Txn(true)
	defer txn.Abort()

	record, ok := findAssetRecord(txn, assetPath)
	if ok {
		if err := txn.Delete(catalogAssetsTable, record); err != nil && !errors.Is(err, memdb.ErrNotFound) {
			panic(err)
		}
	}

	removed := deleteVariantsByAssetPath(txn, assetPath)
	txn.Commit()
	return removed
}

func (c *InMemoryCatalog) DeleteVariants(assetPath string) collectionx.List[*Variant] {
	txn := c.db.Txn(true)
	defer txn.Abort()

	removed := deleteVariantsByAssetPath(txn, assetPath)
	txn.Commit()
	return removed
}

func (c *InMemoryCatalog) DeleteVariantByArtifactPath(artifactPath string) bool {
	txn := c.db.Txn(true)
	defer txn.Abort()

	record, ok := findVariantRecord(txn, catalogVariantArtifactPathIndex, artifactPath)
	if !ok {
		return false
	}
	if err := txn.Delete(catalogVariantsTable, record); err != nil {
		if errors.Is(err, memdb.ErrNotFound) {
			return false
		}
		panic(err)
	}
	txn.Commit()
	return true
}

func (c *InMemoryCatalog) ListVariants(assetPath string) collectionx.List[*Variant] {
	return cloneVariants(c.ListVariantsView(assetPath))
}

func (c *InMemoryCatalog) ListVariantsView(assetPath string) collectionx.List[*Variant] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	return variantViews(txn, "id_prefix", assetPath)
}

func (c *InMemoryCatalog) ListImageVariants(assetPath, format string) collectionx.List[*Variant] {
	return cloneVariants(c.ListImageVariantsView(assetPath, format))
}

func (c *InMemoryCatalog) ListImageVariantsView(assetPath, format string) collectionx.List[*Variant] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	return variantViews(txn, catalogVariantAssetFormatWidthIndex+"_prefix", assetPath, format)
}

func (c *InMemoryCatalog) AllAssets() collectionx.List[*Asset] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(catalogAssetsTable, "id")
	if err != nil {
		panic(err)
	}

	out := collectionx.NewList[*Asset]()
	for raw := iter.Next(); raw != nil; raw = iter.Next() {
		out.Add(cloneAsset(raw.(*assetRecord).Asset))
	}
	return out
}

func (c *InMemoryCatalog) AssetCount() int {
	return countRecords(c.db.Txn(false), catalogAssetsTable)
}

func (c *InMemoryCatalog) VariantCount() int {
	return countRecords(c.db.Txn(false), catalogVariantsTable)
}

func (c *InMemoryCatalog) Snapshot() *Snapshot {
	txn := c.db.Txn(false)
	defer txn.Abort()

	assets, err := txn.Get(catalogAssetsTable, "id")
	if err != nil {
		panic(err)
	}

	entries := collectionx.NewList[*Entry]()
	for raw := assets.Next(); raw != nil; raw = assets.Next() {
		record := raw.(*assetRecord)
		entries.Add(&Entry{
			Asset:    cloneAsset(record.Asset),
			Variants: cloneVariants(variantViews(txn, "id_prefix", record.Path)),
		})
	}
	return &Snapshot{Assets: entries}
}
