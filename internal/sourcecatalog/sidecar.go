package sourcecatalog

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/mo"
	"github.com/samber/oops"
	"golang.org/x/sync/errgroup"
)

type sidecarMatcher struct {
	encoding string
	suffix   string
}

type sidecarFile struct {
	source.File
	assetPath string
	encoding  string
	suffix    string
}

type sidecarVariantBuildCandidate struct {
	sidecar sidecarFile
	asset   *catalog.Asset
}

func IsSourceSidecarVariant(variant *catalog.Variant) bool {
	if variant == nil || variant.Metadata == nil {
		return false
	}
	return strings.TrimSpace(variant.Metadata.GetOrDefault("stage", "")) == SourceSidecarStage
}

func buildSidecarMatchers(registry contentcoding.Registry) collectionx.List[sidecarMatcher] {
	return collectionx.FilterMapList[string, sidecarMatcher](registry.Names(), func(_ int, name string) (sidecarMatcher, bool) {
		strategy, ok := registry.Lookup(name)
		if !ok {
			return sidecarMatcher{}, false
		}
		return sidecarMatcher{
			encoding: strategy.Name(),
			suffix:   strategy.Suffix(),
		}, true
	}).Sort(func(left, right sidecarMatcher) int {
		if len(left.suffix) == len(right.suffix) {
			return cmp.Compare(left.encoding, right.encoding)
		}
		return cmp.Compare(len(right.suffix), len(left.suffix))
	})
}

func recognizeSidecars(filesByPath collectionx.Map[string, source.File], matchers collectionx.List[sidecarMatcher]) collectionx.Map[string, sidecarFile] {
	sidecars := collectionx.NewMapWithCapacity[string, sidecarFile](filesByPath.Len())
	sortedKeys[source.File](filesByPath).Range(func(_ int, path string) bool {
		match, ok := matchSidecar(path, filesByPath, matchers).Get()
		if !ok {
			return true
		}

		match.File = filesByPath.GetOrDefault(path, source.File{})
		sidecars.Set(match.Path, match)
		return true
	})
	return sidecars
}

func matchSidecar(path string, filesByPath collectionx.Map[string, source.File], matchers collectionx.List[sidecarMatcher]) mo.Option[sidecarFile] {
	matcher, ok := collectionx.FindList[sidecarMatcher](matchers, func(_ int, matcher sidecarMatcher) bool {
		if !strings.HasSuffix(path, matcher.suffix) {
			return false
		}
		assetPath := normalizedAssetPath(path, matcher.suffix)
		if assetPath == "" || assetPath == path {
			return false
		}
		_, exists := filesByPath.Get(assetPath)
		return exists
	})
	if !ok {
		return mo.None[sidecarFile]()
	}

	return mo.Some(sidecarFile{
		assetPath: normalizedAssetPath(path, matcher.suffix),
		encoding:  matcher.encoding,
		suffix:    matcher.suffix,
	})
}

func buildSidecarVariants(
	ctx context.Context,
	sidecars collectionx.Map[string, sidecarFile],
	assets collectionx.Map[string, *catalog.Asset],
	existingSidecars collectionx.Map[string, *catalog.Variant],
) (collectionx.Map[string, *catalog.Variant], error) {
	variants, candidates := collectSidecarVariantBuildCandidates(sidecars, assets, existingSidecars)
	if candidates.IsEmpty() {
		return variants, nil
	}

	results := make([]*catalog.Variant, candidates.Len())
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(sourceScanBuildParallelism(candidates.Len()))
	candidates.Range(func(index int, candidate sidecarVariantBuildCandidate) bool {
		group.Go(func() error {
			if err := scanContextErr(groupCtx); err != nil {
				return err
			}
			variant, err := buildSidecarVariant(candidate.sidecar, candidate.asset)
			if err != nil {
				return err
			}
			results[index] = variant
			return nil
		})
		return true
	})
	if err := group.Wait(); err != nil {
		return nil, oops.In("sourcecatalog").Owner("sidecar build").Wrap(err)
	}

	for _, variant := range results {
		variants.Set(variant.ID, variant)
	}
	return variants, nil
}

func collectSidecarVariantBuildCandidates(
	sidecars collectionx.Map[string, sidecarFile],
	assets collectionx.Map[string, *catalog.Asset],
	existingSidecars collectionx.Map[string, *catalog.Variant],
) (collectionx.Map[string, *catalog.Variant], collectionx.List[sidecarVariantBuildCandidate]) {
	variants := collectionx.NewMapWithCapacity[string, *catalog.Variant](sidecars.Len())
	candidates := collectionx.NewList[sidecarVariantBuildCandidate]()

	sortedKeys[sidecarFile](sidecars).Range(func(_ int, sidecarPath string) bool {
		sidecar, _ := sidecars.Get(sidecarPath)
		asset, ok := assets.Get(sidecar.assetPath)
		if !ok || asset == nil {
			return true
		}
		if variant, ok := reusableSidecarVariant(existingSidecars, sidecar, asset).Get(); ok {
			variants.Set(variant.ID, variant)
			return true
		}
		candidates.Add(sidecarVariantBuildCandidate{sidecar: sidecar, asset: asset})
		return true
	})
	return variants, candidates
}

func reusableSidecarVariant(
	existingSidecars collectionx.Map[string, *catalog.Variant],
	sidecar sidecarFile,
	asset *catalog.Asset,
) mo.Option[*catalog.Variant] {
	variant, ok := existingSidecars.Get(sidecar.FullPath)
	if !ok || !canReuseSidecarVariant(variant, sidecar, asset) {
		return mo.None[*catalog.Variant]()
	}
	updateReusableSidecarVariant(variant, sidecar, asset)
	return mo.Some(variant)
}

func updateReusableSidecarVariant(variant *catalog.Variant, sidecar sidecarFile, asset *catalog.Asset) {
	variant.ID = asset.Path + sidecar.suffix
	variant.AssetPath = asset.Path
	variant.ArtifactPath = sidecar.FullPath
	variant.Size = sidecar.Size
	variant.Encoding = sidecar.encoding
	variant.SourceHash = asset.SourceHash
	variant.MediaType = asset.MediaType
	variant.Metadata = catalog.MetadataWithModTime(variant.Metadata, sidecar.ModTime)
}

func canReuseSidecarVariant(variant *catalog.Variant, sidecar sidecarFile, asset *catalog.Asset) bool {
	if variant == nil || asset == nil || !IsSourceSidecarVariant(variant) {
		return false
	}
	modTime, ok := catalog.MetadataModTime(variant.Metadata).Get()
	return ok &&
		variant.ArtifactPath == sidecar.FullPath &&
		variant.Encoding == sidecar.encoding &&
		variant.Size == sidecar.Size &&
		variant.ETag != "" &&
		modTime.Equal(sidecar.ModTime)
}

func buildSidecarVariant(sidecar sidecarFile, asset *catalog.Asset) (*catalog.Variant, error) {
	hash, err := pkg.HashFile(sidecar.FullPath)
	if err != nil {
		return nil, oops.In("sourcecatalog").Owner("variant").With("artifact_path", sidecar.FullPath).Wrap(err)
	}

	return &catalog.Variant{
		ID:           asset.Path + sidecar.suffix,
		AssetPath:    asset.Path,
		ArtifactPath: sidecar.FullPath,
		Size:         sidecar.Size,
		MediaType:    asset.MediaType,
		SourceHash:   asset.SourceHash,
		ETag:         fmt.Sprintf("%q", hash),
		Encoding:     sidecar.encoding,
		Metadata:     sidecarMetadata(sidecar),
	}, nil
}

func normalizedAssetPath(path, suffix string) string {
	return strings.TrimSpace(strings.TrimSuffix(path, suffix))
}

func sidecarMetadata(sidecar sidecarFile) collectionx.Map[string, string] {
	return catalog.MetadataWithModTime(collectionx.NewMapFrom(map[string]string{
		"stage":  SourceSidecarStage,
		"source": filepath.ToSlash(sidecar.Path),
	}), sidecar.ModTime)
}
