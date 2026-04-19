package catalog

import "github.com/hashicorp/go-memdb"

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
					catalogVariantAssetPathIndex: {
						Name:   catalogVariantAssetPathIndex,
						Unique: false,
						Indexer: &memdb.StringFieldIndex{
							Field: "AssetPath",
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
					catalogVariantStageIndex: {
						Name:         catalogVariantStageIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Stage",
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
