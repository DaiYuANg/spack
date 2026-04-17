// Package catalog stores asset and variant metadata in memory.
package catalog

import (
	"errors"
	"strconv"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/media"
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

type assetRecord struct {
	Path  string
	Asset *Asset
}

type variantRecord struct {
	AssetPath    string
	ID           string
	ArtifactPath string
	Encoding     string
	ImageFormat  string
	Width        int
	Variant      *Variant
}

func NewCatalog() Catalog {
	return NewInMemoryCatalog()
}

func NewInMemoryCatalog() *InMemoryCatalog {
	db, err := memdb.NewMemDB(newCatalogSchema())
	if err != nil {
		panic(err)
	}
	return &InMemoryCatalog{db: db}
}

func newCatalogSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			catalogAssetsTable: {
				Name: catalogAssetsTable,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Path",
						},
					},
				},
			},
			catalogVariantsTable: {
				Name: catalogVariantsTable,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "AssetPath"},
								&memdb.StringFieldIndex{Field: "ID"},
							},
						},
					},
					catalogVariantArtifactPathIndex: {
						Name:         catalogVariantArtifactPathIndex,
						Unique:       true,
						AllowMissing: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "ArtifactPath",
						},
					},
					catalogVariantAssetEncodingIndex: {
						Name:         catalogVariantAssetEncodingIndex,
						Unique:       true,
						AllowMissing: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "AssetPath"},
								&memdb.StringFieldIndex{Field: "Encoding"},
							},
						},
					},
					catalogVariantAssetFormatWidthIndex: {
						Name:         catalogVariantAssetFormatWidthIndex,
						Unique:       true,
						AllowMissing: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "AssetPath"},
								&memdb.StringFieldIndex{Field: "ImageFormat"},
								&memdb.IntFieldIndex{Field: "Width"},
							},
						},
					},
				},
			},
		},
	}
}

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
