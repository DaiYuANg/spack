package catalog

import (
	"errors"
	"strconv"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/media"
	"github.com/hashicorp/go-memdb"
)

func cloneAsset(asset *Asset) *Asset {
	if asset == nil {
		return nil
	}

	cloned := *asset
	cloned.Metadata = CloneMetadata(asset.Metadata)
	return &cloned
}

func cloneVariant(variant *Variant) *Variant {
	if variant == nil {
		return nil
	}

	cloned := *variant
	cloned.Metadata = CloneMetadata(variant.Metadata)
	return &cloned
}

func cloneVariants(variants collectionx.List[*Variant]) collectionx.List[*Variant] {
	return collectionx.MapList(variants, func(_ int, variant *Variant) *Variant {
		return cloneVariant(variant)
	})
}

func prepareAsset(asset *Asset) *Asset {
	cloned := cloneAsset(asset)
	if cloned == nil {
		return nil
	}
	cloned.Metadata = EnsureMetadataModTime(cloned.Metadata, cloned.FullPath)
	return cloned
}

func prepareVariant(variant *Variant) *Variant {
	cloned := cloneVariant(variant)
	if cloned == nil {
		return nil
	}
	cloned.Metadata = EnsureMetadataModTime(cloned.Metadata, cloned.ArtifactPath)
	return cloned
}

func newAssetRecord(asset *Asset) *assetRecord {
	prepared := prepareAsset(asset)
	return &assetRecord{
		Path:  prepared.Path,
		Asset: prepared,
	}
}

func newVariantRecord(variant *Variant, id string) *variantRecord {
	prepared := prepareVariant(variant)
	prepared.ID = id
	return &variantRecord{
		AssetPath:    prepared.AssetPath,
		ID:           prepared.ID,
		ArtifactPath: prepared.ArtifactPath,
		Encoding:     prepared.Encoding,
		ImageFormat:  imageFormatKey(prepared),
		Width:        prepared.Width,
		Variant:      prepared,
	}
}

func imageFormatKey(variant *Variant) string {
	if variant == nil {
		return ""
	}
	if variant.Format != "" {
		return variant.Format
	}
	return media.ImageFormat(variant.MediaType)
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

func collectVariantDelete(txn *memdb.Txn, pending map[string]*variantRecord, index string, args ...interface{}) {
	record, ok := findVariantRecord(txn, index, args...)
	if !ok {
		return
	}
	pending[variantRecordKey(record)] = record
}

func variantRecordKey(record *variantRecord) string {
	if record == nil {
		return ""
	}
	return record.AssetPath + "\x00" + record.ID
}

func deleteVariantsByAssetPath(txn *memdb.Txn, assetPath string) collectionx.List[*Variant] {
	records := variantRecords(txn, "id_prefix", assetPath)
	removed := cloneVariantRecords(records)
	records.Range(func(_ int, record *variantRecord) bool {
		if err := txn.Delete(catalogVariantsTable, record); err != nil && !errors.Is(err, memdb.ErrNotFound) {
			panic(err)
		}
		return true
	})
	return removed
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

func cloneVariantRecords(records collectionx.List[*variantRecord]) collectionx.List[*Variant] {
	return collectionx.MapList(records, func(_ int, record *variantRecord) *Variant {
		return cloneVariant(record.Variant)
	})
}
