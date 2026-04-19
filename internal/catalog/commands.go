package catalog

import (
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/hashicorp/go-memdb"
)

func (c *MemDBCatalog) UpsertAsset(asset *Asset) error {
	record := newAssetRecord(asset)

	txn := c.db.Txn(true)
	defer txn.Abort()

	if err := txn.Insert(catalogAssetsTable, record); err != nil {
		return err
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

func (c *MemDBCatalog) DeleteAsset(assetPath string) collectionx.List[*Variant] {
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

func collectVariantDelete(txn *memdb.Txn, pending map[string]*variantRecord, index string, args ...interface{}) {
	record, ok := findVariantRecord(txn, index, args...)
	if !ok {
		return
	}
	pending[variantRecordKey(record)] = record
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
