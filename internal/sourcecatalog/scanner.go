package sourcecatalog

import (
	"context"

	"github.com/arcgolabs/collectionx"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/source"
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

func NewScanner(src source.Source, registry contentcoding.Registry) Scanner {
	return Scanner{
		src:      src,
		matchers: buildSidecarMatchers(registry),
	}
}

func (s Scanner) Scan(ctx context.Context) (Snapshot, error) {
	return s.ScanWithCatalog(ctx, nil)
}

func (s Scanner) Watch(ctx context.Context) (<-chan source.ChangeEvent, error) {
	watcher, ok := s.src.(source.Watcher)
	if !ok {
		return nil, source.ErrWatchUnsupported
	}
	changes, err := watcher.Watch(ctx)
	if err != nil {
		return nil, oops.In("sourcecatalog").Owner("watch").Wrap(err)
	}
	return changes, nil
}

func (s Scanner) ScanWithCatalog(ctx context.Context, cat catalog.Catalog) (Snapshot, error) {
	ctx = normalizeScanContext(ctx)

	filesByPath, totalBytes, err := s.collectSourceFiles(ctx)
	if err != nil {
		return Snapshot{}, err
	}

	existing := buildExistingScanState(cat)
	sidecars := recognizeSidecars(filesByPath, s.matchers)
	assets, err := buildAssets(ctx, filesByPath, sidecars, existing.assets)
	if err != nil {
		return Snapshot{}, err
	}
	variants, err := buildSidecarVariants(ctx, sidecars, assets, existing.sidecars)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Assets:     assets,
		Variants:   variants,
		TotalBytes: totalBytes,
	}, nil
}

func (s Scanner) collectSourceFiles(ctx context.Context) (collectionx.Map[string, source.File], int64, error) {
	scanErr := oops.In("sourcecatalog").Owner("scan")
	filesByPath := collectionx.NewMap[string, source.File]()
	totalBytes := int64(0)

	if err := s.src.Walk(func(file source.File) error {
		if err := scanContextErr(ctx); err != nil {
			return err
		}
		if file.IsDir {
			return nil
		}
		filesByPath.Set(file.Path, file)
		totalBytes += file.Size
		return nil
	}); err != nil {
		return nil, 0, scanErr.Wrap(err)
	}

	return filesByPath, totalBytes, nil
}

func normalizeScanContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func scanContextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return oops.In("sourcecatalog").Owner("scan context").Wrap(err)
	}
	return nil
}
