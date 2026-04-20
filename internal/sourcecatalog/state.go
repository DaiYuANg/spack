package sourcecatalog

import (
	"cmp"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
)

type existingScanState struct {
	assets   collectionx.Map[string, *catalog.Asset]
	sidecars collectionx.Map[string, *catalog.Variant]
}

func buildExistingScanState(cat catalog.Catalog) existingScanState {
	state := existingScanState{
		assets:   collectionx.NewMap[string, *catalog.Asset](),
		sidecars: collectionx.NewMap[string, *catalog.Variant](),
	}
	if cat == nil {
		return state
	}

	assets := cat.AllAssets()
	state.assets = collectionx.AssociateList[*catalog.Asset, string, *catalog.Asset](assets, func(_ int, asset *catalog.Asset) (string, *catalog.Asset) {
		return asset.Path, asset
	})
	cat.ListVariantsByStage(SourceSidecarStage).Range(func(_ int, variant *catalog.Variant) bool {
		if IsSourceSidecarVariant(variant) {
			state.sidecars.Set(variant.ArtifactPath, variant)
		}
		return true
	})
	return state
}

func sortedKeys[T any](values collectionx.Map[string, T]) collectionx.List[string] {
	return collectionx.NewList[string](values.Keys()...).Sort(cmp.Compare[string])
}
