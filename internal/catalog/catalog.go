// Package catalog stores asset and variant metadata in memory.
package catalog

import "errors"

const (
	catalogAssetsTable                  = "assets"
	catalogVariantsTable                = "variants"
	catalogVariantAssetPathIndex        = "asset_path"
	catalogVariantArtifactPathIndex     = "artifact_path"
	catalogVariantAssetEncodingIndex    = "asset_path_encoding"
	catalogVariantAssetFormatWidthIndex = "asset_path_format_width"
	catalogVariantStageIndex            = "stage"
)

var (
	ErrAssetNotFound       = errors.New("asset not found")
	ErrRecordTypeMismatch  = errors.New("catalog record type mismatch")
	ErrCatalogQueryFailure = errors.New("catalog query failed")
)
