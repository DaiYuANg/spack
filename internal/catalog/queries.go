package catalog

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/hashicorp/go-memdb"
)

func (c *MemDBCatalog) FindAsset(assetPath string) (*Asset, bool) {
	asset, ok := c.FindAssetView(assetPath)
	return cloneAsset(asset), ok
}

func (c *MemDBCatalog) FindAssetView(assetPath string) (*Asset, bool) {
	txn := c.db.Txn(false)
	defer txn.Abort()

	record, ok := findAssetRecord(txn, assetPath)
	if !ok {
		return nil, false
	}
	return record.Asset, true
}

func (c *MemDBCatalog) FindEncodingVariant(assetPath, encoding string) (*Variant, bool) {
	variant, ok := c.FindEncodingVariantView(assetPath, encoding)
	return cloneVariant(variant), ok
}

func (c *MemDBCatalog) FindEncodingVariantView(assetPath, encoding string) (*Variant, bool) {
	txn := c.db.Txn(false)
	defer txn.Abort()

	record, ok := findVariantRecord(txn, catalogVariantAssetEncodingIndex, assetPath, encoding)
	if !ok {
		return nil, false
	}
	return record.Variant, true
}

func (c *MemDBCatalog) FindImageVariant(assetPath, format string, width int) (*Variant, bool) {
	variant, ok := c.FindImageVariantView(assetPath, format, width)
	return cloneVariant(variant), ok
}

func (c *MemDBCatalog) FindImageVariantView(assetPath, format string, width int) (*Variant, bool) {
	txn := c.db.Txn(false)
	defer txn.Abort()

	record, ok := findVariantRecord(txn, catalogVariantAssetFormatWidthIndex, assetPath, format, width)
	if !ok {
		return nil, false
	}
	return record.Variant, true
}

func (c *MemDBCatalog) ListVariants(assetPath string) collectionx.List[*Variant] {
	return cloneVariants(c.ListVariantsView(assetPath))
}

func (c *MemDBCatalog) ListVariantsView(assetPath string) collectionx.List[*Variant] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	return variantViews(txn, catalogVariantAssetPathIndex, assetPath)
}

func (c *MemDBCatalog) ListImageVariants(assetPath, format string) collectionx.List[*Variant] {
	return cloneVariants(c.ListImageVariantsView(assetPath, format))
}

func (c *MemDBCatalog) ListImageVariantsView(assetPath, format string) collectionx.List[*Variant] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	return variantViews(txn, catalogVariantAssetFormatWidthIndex+"_prefix", assetPath, format)
}

func (c *MemDBCatalog) ListVariantsByStage(stage string) collectionx.List[*Variant] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	return cloneVariants(variantViews(txn, catalogVariantStageIndex, stage))
}

func (c *MemDBCatalog) AllAssets() collectionx.List[*Asset] {
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

func (c *MemDBCatalog) AllVariants() collectionx.List[*Variant] {
	txn := c.db.Txn(false)
	defer txn.Abort()

	return cloneVariants(variantViews(txn, "id"))
}

func (c *MemDBCatalog) AssetCount() int {
	return countRecords(c.db.Txn(false), catalogAssetsTable)
}

func (c *MemDBCatalog) VariantCount() int {
	return countRecords(c.db.Txn(false), catalogVariantsTable)
}

func (c *MemDBCatalog) Snapshot() *Snapshot {
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
			Variants: cloneVariants(variantViews(txn, catalogVariantAssetPathIndex, record.Path)),
		})
	}
	return &Snapshot{Assets: entries}
}

func assetExists(txn *memdb.Txn, assetPath string) bool {
	_, ok := findAssetRecord(txn, assetPath)
	return ok
}

func findAssetRecord(txn *memdb.Txn, assetPath string) (*assetRecord, bool) {
	raw, err := txn.First(catalogAssetsTable, "id", assetPath)
	if err != nil || raw == nil {
		return nil, false
	}
	return raw.(*assetRecord), true
}

func findVariantRecord(txn *memdb.Txn, index string, args ...interface{}) (*variantRecord, bool) {
	raw, err := txn.First(catalogVariantsTable, index, args...)
	if err != nil || raw == nil {
		return nil, false
	}
	return raw.(*variantRecord), true
}

func countRecords(txn *memdb.Txn, table string) int {
	defer txn.Abort()

	iter, err := txn.Get(table, "id")
	if err != nil {
		panic(err)
	}

	total := 0
	for raw := iter.Next(); raw != nil; raw = iter.Next() {
		total++
	}
	return total
}

func variantViews(txn *memdb.Txn, index string, args ...interface{}) collectionx.List[*Variant] {
	records := variantRecords(txn, index, args...)
	return collectionx.MapList(records, func(_ int, record *variantRecord) *Variant {
		return record.Variant
	})
}

func variantRecords(txn *memdb.Txn, index string, args ...interface{}) collectionx.List[*variantRecord] {
	iter, err := txn.Get(catalogVariantsTable, index, args...)
	if err != nil {
		panic(err)
	}

	out := collectionx.NewList[*variantRecord]()
	for raw := iter.Next(); raw != nil; raw = iter.Next() {
		out.Add(raw.(*variantRecord))
	}
	return out
}
