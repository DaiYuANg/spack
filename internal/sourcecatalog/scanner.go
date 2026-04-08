package sourcecatalog

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/samber/oops"
)

const SourceSidecarStage = "source_sidecar"

type Snapshot struct {
	Assets     collectionx.Map[string, *catalog.Asset]
	Variants   collectionx.Map[string, *catalog.Variant]
	TotalBytes int64
}

type Scanner struct {
	src      source.Source
	matchers collectionx.List[sidecarMatcher]
}

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

func NewScanner(src source.Source, registry contentcoding.Registry) Scanner {
	return Scanner{
		src:      src,
		matchers: buildSidecarMatchers(registry),
	}
}

func (s Scanner) Scan(ctx context.Context) (Snapshot, error) {
	scanErr := oops.In("sourcecatalog").Owner("scan")
	filesByPath := collectionx.NewMap[string, source.File]()
	totalBytes := int64(0)

	if err := s.src.Walk(func(file source.File) error {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if file.IsDir {
			return nil
		}
		filesByPath.Set(file.Path, file)
		totalBytes += file.Size
		return nil
	}); err != nil {
		return Snapshot{}, scanErr.Wrap(err)
	}

	sidecars := recognizeSidecars(filesByPath, s.matchers)
	assets, err := buildAssets(filesByPath, sidecars)
	if err != nil {
		return Snapshot{}, err
	}
	variants, err := buildSidecarVariants(sidecars, assets)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Assets:     assets,
		Variants:   variants,
		TotalBytes: totalBytes,
	}, nil
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

func IsSourceSidecarVariant(variant *catalog.Variant) bool {
	if variant == nil || variant.Metadata == nil {
		return false
	}
	return strings.TrimSpace(variant.Metadata.GetOrDefault("stage", "")) == SourceSidecarStage
}

func buildSidecarMatchers(registry contentcoding.Registry) collectionx.List[sidecarMatcher] {
	return collectionx.NewList(lo.FilterMap(registry.Names().Values(), func(name string, _ int) (sidecarMatcher, bool) {
		strategy, ok := registry.Lookup(name)
		if !ok {
			return sidecarMatcher{}, false
		}
		return sidecarMatcher{
			encoding: strategy.Name(),
			suffix:   strategy.Suffix(),
		}, true
	})...).Sort(func(left, right sidecarMatcher) int {
		if len(left.suffix) == len(right.suffix) {
			return cmp.Compare(left.encoding, right.encoding)
		}
		return cmp.Compare(len(right.suffix), len(left.suffix))
	})
}

func recognizeSidecars(filesByPath collectionx.Map[string, source.File], matchers collectionx.List[sidecarMatcher]) collectionx.Map[string, sidecarFile] {
	sidecars := collectionx.NewMapWithCapacity[string, sidecarFile](filesByPath.Len())
	sortedKeys(filesByPath).Range(func(_ int, path string) bool {
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
	matcher, ok := lo.Find(matchers.Values(), func(matcher sidecarMatcher) bool {
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

func buildAssets(filesByPath collectionx.Map[string, source.File], sidecars collectionx.Map[string, sidecarFile]) (collectionx.Map[string, *catalog.Asset], error) {
	assets := collectionx.NewMapWithCapacity[string, *catalog.Asset](filesByPath.Len())
	var buildErr error
	sortedKeys(filesByPath).Range(func(_ int, path string) bool {
		if _, ok := sidecars.Get(path); ok {
			return true
		}
		file, _ := filesByPath.Get(path)
		asset, err := BuildAsset(file)
		if err != nil {
			buildErr = err
			return false
		}
		assets.Set(path, asset)
		return true
	})
	if buildErr != nil {
		return nil, buildErr
	}
	return assets, nil
}

func buildSidecarVariants(sidecars collectionx.Map[string, sidecarFile], assets collectionx.Map[string, *catalog.Asset]) (collectionx.Map[string, *catalog.Variant], error) {
	variants := collectionx.NewMapWithCapacity[string, *catalog.Variant](sidecars.Len())
	var buildErr error
	sortedKeys(sidecars).Range(func(_ int, sidecarPath string) bool {
		sidecar, _ := sidecars.Get(sidecarPath)
		asset, ok := assets.Get(sidecar.assetPath)
		if !ok || asset == nil {
			return true
		}
		variant, err := buildSidecarVariant(sidecar, asset)
		if err != nil {
			buildErr = err
			return false
		}
		variants.Set(variant.ID, variant)
		return true
	})
	if buildErr != nil {
		return nil, buildErr
	}
	return variants, nil
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

func sortedKeys[T any](values collectionx.Map[string, T]) collectionx.List[string] {
	return collectionx.NewList(values.Keys()...).Sort(cmp.Compare[string])
}

func normalizedAssetPath(path, suffix string) string {
	return strings.TrimSpace(strings.TrimSuffix(path, suffix))
}

func assetMetadata(file source.File) collectionx.Map[string, string] {
	metadata := collectionx.NewMapWithCapacity[string, string](1)
	metadata.Set("mtime_unix", strconv.FormatInt(file.ModTime.Unix(), 10))
	return metadata
}

func sidecarMetadata(sidecar sidecarFile) collectionx.Map[string, string] {
	metadata := collectionx.NewMapWithCapacity[string, string](3)
	metadata.Set("stage", SourceSidecarStage)
	metadata.Set("mtime_unix", strconv.FormatInt(sidecar.ModTime.Unix(), 10))
	metadata.Set("source", filepath.ToSlash(sidecar.Path))
	return metadata
}
