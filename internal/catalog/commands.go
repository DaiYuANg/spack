package catalog

import (
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/hashicorp/go-memdb"
	"github.com/samber/oops"
)

func (c *MemDBCatalog) UpsertAsset(asset *Asset) error {
	record := newAssetRecord(asset)

	txn := c.db.Txn(true)
	defer txn.Abort()

	if err := txn.Insert(catalogAssetsTable, record); err != nil {
		return oops.In("catalog").Owner("asset upsert").Wrap(err)
	}
	txn.Commit()
	return nil
}

func (c *MemDBCatalog) UpsertVariant(variant *Variant) error {
	id := variant.ID
	if id == "" {
		id = defaultVariantID(variant)
	}

	record := newVariantRecord(variant, id)
	txn := c.db.Txn(true)
	defer txn.Abort()

	exists, err := assetExists(txn, record.AssetPath)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAssetNotFound
	}

	pendingDeletes, err := collectPendingVariantDeletes(txn, record)
	if err != nil {
		return err
	}

	for _, existing := range pendingDeletes.Values() {
		if err := txn.Delete(catalogVariantsTable, existing); err != nil && !errors.Is(err, memdb.ErrNotFound) {
			return oops.In("catalog").Owner("variant upsert").Wrap(err)
		}
	}
	if err := txn.Insert(catalogVariantsTable, record); err != nil {
		return oops.In("catalog").Owner("variant upsert").Wrap(err)
	}
	txn.Commit()
	return nil
}

func (c *MemDBCatalog) DeleteAsset(assetPath string) collectionx.List[*Variant] {
	txn := c.db.Txn(true)
	defer txn.Abort()

	record, ok, err := findAssetRecord(txn, assetPath)
	if err != nil {
		return collectionx.NewList[*Variant]()
	}
	if ok {
		if err := txn.Delete(catalogAssetsTable, record); err != nil && !errors.Is(err, memdb.ErrNotFound) {
			panic(err)
		}
	}

	removed := deleteVariantsByAssetPath(txn, assetPath)
	txn.Commit()
	return removed
}

func (c *MemDBCatalog) DeleteVariants(assetPath string) collectionx.List[*Variant] {
	txn := c.db.Txn(true)
	defer txn.Abort()

	removed := deleteVariantsByAssetPath(txn, assetPath)
	txn.Commit()
	return removed
}

func (c *MemDBCatalog) DeleteVariantByArtifactPath(artifactPath string) bool {
	txn := c.db.Txn(true)
	defer txn.Abort()

	record, ok, err := findVariantRecord(txn, catalogVariantArtifactPathIndex, artifactPath)
	if err != nil || !ok {
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

func collectPendingVariantDeletes(txn *memdb.Txn, record *variantRecord) (collectionx.Map[string, *variantRecord], error) {
	pendingDeletes := collectionx.NewMapWithCapacity[string, *variantRecord](4)
	lookups := variantDeleteLookups(record)
	for _, lookup := range lookups {
		if err := collectVariantDelete(txn, pendingDeletes, lookup.index, lookup.args...); err != nil {
			return nil, err
		}
	}
	return pendingDeletes, nil
}

type variantDeleteLookup struct {
	index string
	args  []any
}

func variantDeleteLookups(record *variantRecord) []variantDeleteLookup {
	lookups := []variantDeleteLookup{
		{index: "id", args: []any{record.AssetPath, record.ID}},
	}
	if record.ArtifactPath != "" {
		lookups = append(lookups, variantDeleteLookup{index: catalogVariantArtifactPathIndex, args: []any{record.ArtifactPath}})
	}
	if record.Encoding != "" {
		lookups = append(lookups, variantDeleteLookup{index: catalogVariantAssetEncodingIndex, args: []any{record.AssetPath, record.Encoding}})
	}
	if record.ImageFormat != "" {
		lookups = append(lookups, variantDeleteLookup{index: catalogVariantAssetFormatWidthIndex, args: []any{record.AssetPath, record.ImageFormat, record.Width}})
	}
	return lookups
}

func collectVariantDelete(txn *memdb.Txn, pending collectionx.Map[string, *variantRecord], index string, args ...any) error {
	record, ok, err := findVariantRecord(txn, index, args...)
	if err != nil || !ok {
		return err
	}
	pending.Set(variantRecordKey(record), record)
	return nil
}

func deleteVariantsByAssetPath(txn *memdb.Txn, assetPath string) collectionx.List[*Variant] {
	records := variantRecords(txn, catalogVariantAssetPathIndex, assetPath)
	removed := cloneVariantRecords(records)
	records.Range(func(_ int, record *variantRecord) bool {
		if err := txn.Delete(catalogVariantsTable, record); err != nil && !errors.Is(err, memdb.ErrNotFound) {
			panic(err)
		}
		return true
	})
	return removed
}
