package sourcecatalog

import (
	"context"
	"fmt"
	"runtime"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/oops"
	"golang.org/x/sync/errgroup"
)

const maxSourceScanBuildParallelism = 16

type assetBuildCandidate struct {
	path string
	file source.File
}

type assetBuildResult struct {
	path  string
	asset *catalog.Asset
}

func BuildAsset(file source.File) (*catalog.Asset, error) {
	sourceHash, err := pkg.HashFile(file.FullPath)
	if err != nil {
		return nil, oops.In("sourcecatalog").Owner("asset").With("asset_path", file.Path).Wrap(err)
	}
	return &catalog.Asset{
		Path:       file.Path,
		FullPath:   file.FullPath,
		Size:       file.Size,
		MediaType:  string(pkg.DetectMIME(file.FullPath)),
		SourceHash: sourceHash,
		ETag:       fmt.Sprintf("%q", sourceHash),
		Metadata:   assetMetadata(file),
	}, nil
}

func buildAssets(
	ctx context.Context,
	filesByPath collectionx.Map[string, source.File],
	sidecars collectionx.Map[string, sidecarFile],
	existingAssets collectionx.Map[string, *catalog.Asset],
) (collectionx.Map[string, *catalog.Asset], error) {
	assets, candidates := collectAssetBuildCandidates(filesByPath, sidecars, existingAssets)
	if candidates.IsEmpty() {
		return assets, nil
	}

	results := make([]assetBuildResult, candidates.Len())
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(sourceScanBuildParallelism(candidates.Len()))
	candidates.Range(func(index int, candidate assetBuildCandidate) bool {
		group.Go(func() error {
			if err := scanContextErr(groupCtx); err != nil {
				return err
			}
			asset, err := BuildAsset(candidate.file)
			if err != nil {
				return err
			}
			results[index] = assetBuildResult{path: candidate.path, asset: asset}
			return nil
		})
		return true
	})
	if err := group.Wait(); err != nil {
		return nil, oops.In("sourcecatalog").Owner("asset build").Wrap(err)
	}

	for _, result := range results {
		assets.Set(result.path, result.asset)
	}
	return assets, nil
}

func collectAssetBuildCandidates(
	filesByPath collectionx.Map[string, source.File],
	sidecars collectionx.Map[string, sidecarFile],
	existingAssets collectionx.Map[string, *catalog.Asset],
) (collectionx.Map[string, *catalog.Asset], collectionx.List[assetBuildCandidate]) {
	assets := collectionx.NewMapWithCapacity[string, *catalog.Asset](filesByPath.Len())
	candidates := collectionx.NewList[assetBuildCandidate]()

	sortedKeys[source.File](filesByPath).Range(func(_ int, path string) bool {
		if _, ok := sidecars.Get(path); ok {
			return true
		}
		file, _ := filesByPath.Get(path)
		if asset, ok := existingAssets.Get(path); ok && canReuseAsset(asset, file) {
			asset.Metadata = catalog.MetadataWithModTime(asset.Metadata, file.ModTime)
			assets.Set(path, asset)
			return true
		}
		candidates.Add(assetBuildCandidate{path: path, file: file})
		return true
	})
	return assets, candidates
}

func canReuseAsset(asset *catalog.Asset, file source.File) bool {
	if asset == nil {
		return false
	}
	modTime, ok := catalog.MetadataModTime(asset.Metadata).Get()
	return ok &&
		asset.Path == file.Path &&
		asset.FullPath == file.FullPath &&
		asset.Size == file.Size &&
		asset.MediaType != "" &&
		asset.SourceHash != "" &&
		asset.ETag != "" &&
		modTime.Equal(file.ModTime)
}

func sourceScanBuildParallelism(total int) int {
	if total < 2 {
		return 1
	}
	return min(total, max(1, min(maxSourceScanBuildParallelism, runtime.GOMAXPROCS(0)*2)))
}

func assetMetadata(file source.File) collectionx.Map[string, string] {
	return catalog.MetadataWithModTime(collectionx.NewMap[string, string](), file.ModTime)
}
