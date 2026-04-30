package catalog

import (
	"strconv"

	cxlist "github.com/arcgolabs/collectionx/list"
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

func cloneVariants(variants *cxlist.List[*Variant]) *cxlist.List[*Variant] {
	return cxlist.MapList[*Variant, *Variant](variants, func(_ int, variant *Variant) *Variant {
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

func cloneVariantRecords(records *cxlist.List[*variantRecord]) *cxlist.List[*Variant] {
	return cxlist.MapList[*variantRecord, *Variant](records, func(_ int, record *variantRecord) *Variant {
		return cloneVariant(record.Variant)
	})
}
