package task

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/samber/oops"
)

func openArtifactRoot(root string) (*os.Root, error) {
	rootHandle, err := os.OpenRoot(root)
	if err == nil {
		return rootHandle, nil
	}
	if os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	return nil, oops.In("task").Owner("artifact janitor").Wrap(err)
}

func visitArtifactPath(
	ctx context.Context,
	rootHandle *os.Root,
	root string,
	relativePath string,
	entry fs.DirEntry,
	walkErr error,
	expected collectionx.Set[string],
	cat catalog.Catalog,
	bodyCache *assetcache.Cache,
	report *ArtifactJanitorReport,
) error {
	if walkErr != nil {
		return walkErr
	}
	if ctxErr := janitorContextErr(ctx); ctxErr != nil {
		return ctxErr
	}
	if entry.IsDir() {
		return nil
	}

	artifactPath := filepath.Join(root, filepath.FromSlash(relativePath))
	if report != nil {
		report.ScannedArtifacts++
	}
	if expected != nil && expected.Contains(artifactPath) {
		return nil
	}
	return removeOrphanArtifact(rootHandle, relativePath, artifactPath, cat, bodyCache, report)
}

func artifactExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func variantArtifactPath(variant *catalog.Variant) string {
	if variant == nil {
		return ""
	}
	return strings.TrimSpace(variant.ArtifactPath)
}

func janitorContextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return oops.In("task").Owner("artifact janitor").Wrap(err)
	}
	return nil
}

func closeRoot(root *os.Root) error {
	if root == nil {
		return nil
	}
	if err := root.Close(); err != nil {
		return oops.In("task").Owner("artifact janitor").Wrap(err)
	}
	return nil
}
